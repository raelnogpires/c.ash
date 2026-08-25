package importer

import (
	"encoding/xml"
	"errors"
	"html"
	"io"
	"regexp"
	"strings"
	"time"

	"c.ash/internal/domain"
)

var (
	ofxRoot        = regexp.MustCompile(`(?i)<OFX(?:\s|>)`)
	ofxXMLDecl     = regexp.MustCompile(`(?is)<\?xml\b[^?]*\?>`)
	ofxVersion2    = regexp.MustCompile(`(?im)^\s*VERSION\s*:\s*2`)
	ofxTxnStart    = regexp.MustCompile(`(?i)<STMTTRN(?:\s|>)`)
	ofxTransaction = regexp.MustCompile(`(?is)<STMTTRN\b[^>]*>(.*?)</STMTTRN\s*>`)
)

// ParseOFX extracts posted banking transactions from OFX 1.x SGML and OFX 2.x
// XML documents. Institution metadata is deliberately ignored; provenance is
// supplied by the bank selected by the user.
func ParseOFX(data []byte) ([]Entry, error) {
	text, err := decodeStatementText(data)
	if err != nil {
		return nil, err
	}
	if !ofxRoot.MatchString(text) {
		return nil, domain.ErrInvalidStatement
	}

	var records []ofxRecord
	if ofxXMLDecl.MatchString(text) || ofxVersion2.MatchString(text) || strings.Contains(strings.ToUpper(text), "</TRNAMT>") {
		records, err = parseXMLOFX(text)
	} else {
		records, err = parseSGMLOFX(text)
	}
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(records))
	for _, record := range records {
		entry, parseErr := record.entry()
		if parseErr != nil {
			return nil, domain.ErrInvalidStatement
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, domain.ErrStatementEmpty
	}
	return entries, nil
}

type ofxRecord struct {
	Date   string
	Amount string
	Name   string
	Memo   string
}

func (record ofxRecord) entry() (Entry, error) {
	rawDate := strings.TrimSpace(record.Date)
	if len(rawDate) < 8 {
		return Entry{}, errors.New("missing OFX date")
	}
	date, err := time.Parse("20060102", rawDate[:8])
	if err != nil {
		return Entry{}, err
	}
	amount, _, err := parseMoneyCents(record.Amount)
	if err != nil || amount == 0 {
		return Entry{}, errors.New("invalid OFX amount")
	}
	description := strings.TrimSpace(record.Name)
	memo := strings.TrimSpace(record.Memo)
	if description == "" {
		description = memo
	}
	if description == "" {
		return Entry{}, errors.New("missing OFX description")
	}
	kind := domain.Income
	if amount < 0 {
		kind = domain.Expense
		amount = -amount
	}
	return Entry{Kind: kind, AmountCents: amount, Description: cleanLine(html.UnescapeString(description)), Date: date.Format("2006-01-02")}, nil
}

func parseXMLOFX(text string) ([]ofxRecord, error) {
	root := ofxRoot.FindStringIndex(text)
	if root == nil {
		return nil, domain.ErrInvalidStatement
	}
	// OFX 2 may retain its HTTP-style header before the XML body. Parsing from
	// the root also avoids conflicts with a legacy encoding declaration after
	// the bytes have already been converted to UTF-8.
	text = text[root[0]:]
	decoder := xml.NewDecoder(strings.NewReader(text))
	var records []ofxRecord
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, domain.ErrInvalidStatement
		}
		start, ok := token.(xml.StartElement)
		if !ok || !strings.EqualFold(start.Name.Local, "STMTTRN") {
			continue
		}
		record, err := readXMLTransaction(decoder, start)
		if err != nil {
			return nil, domain.ErrInvalidStatement
		}
		records = append(records, record)
	}
	return records, nil
}

func readXMLTransaction(decoder *xml.Decoder, start xml.StartElement) (ofxRecord, error) {
	var record ofxRecord
	depth := 1
	field := ""
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return record, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 {
				field = strings.ToUpper(value.Name.Local)
			}
		case xml.EndElement:
			if depth == 2 {
				field = ""
			}
			depth--
		case xml.CharData:
			if depth != 2 {
				continue
			}
			switch field {
			case "DTPOSTED":
				record.Date += string(value)
			case "TRNAMT":
				record.Amount += string(value)
			case "NAME":
				record.Name += string(value)
			case "MEMO":
				record.Memo += string(value)
			}
		}
	}
	return record, nil
}

func parseSGMLOFX(text string) ([]ofxRecord, error) {
	upper := strings.ToUpper(text)
	if !strings.Contains(upper, "</OFX>") {
		return nil, domain.ErrInvalidStatement
	}
	matches := ofxTransaction.FindAllStringSubmatch(text, -1)
	transactionCount := len(ofxTxnStart.FindAllStringIndex(text, -1))
	if transactionCount != len(matches) {
		return nil, domain.ErrInvalidStatement
	}
	if len(matches) == 0 {
		return nil, nil
	}
	records := make([]ofxRecord, 0, len(matches))
	for _, match := range matches {
		records = append(records, ofxRecord{
			Date:   sgmlValue(match[1], "DTPOSTED"),
			Amount: sgmlValue(match[1], "TRNAMT"),
			Name:   sgmlValue(match[1], "NAME"),
			Memo:   sgmlValue(match[1], "MEMO"),
		})
	}
	return records, nil
}

func sgmlValue(block, tag string) string {
	pattern := regexp.MustCompile(`(?is)<` + tag + `\b[^>]*>\s*([^<\r\n]*)`)
	match := pattern.FindStringSubmatch(block)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

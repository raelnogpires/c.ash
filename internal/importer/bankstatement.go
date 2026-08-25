// Package importer extracts transactions from supported Brazilian bank statements.
package importer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"c.ash/internal/domain"
	"github.com/ledongthuc/pdf"
)

const MaxStatementSize = 15 << 20

// MaxPDFSize is kept as an alias for callers compiled against the original
// PDF-only importer.
const MaxPDFSize = MaxStatementSize

type Bank string

const (
	BankItau     Bank = "itau"
	BankBradesco Bank = "bradesco"
	BankInter    Bank = "inter"
)

type Entry struct {
	Kind        domain.TransactionKind
	AmountCents int64
	Description string
	Date        string
}

var (
	dateAtStart  = regexp.MustCompile(`^\s*(\d{2})[/-](\d{2})(?:[/-](\d{2}|\d{4}))?\b\s*`)
	dateAnywhere = regexp.MustCompile(`\d{2}[/-]\d{2}[/-]\d{4}\b`)
	fullDate     = regexp.MustCompile(`\b\d{2}[/-]\d{2}[/-](\d{4})\b`)
	amountAtEnd  = regexp.MustCompile(`(?i)(?:R\$\s*)?([+-]?\s*(?:\d{1,3}(?:\.\d{3})*|\d+),\d{2})\s*([DC]|[-+])?\s*$`)
	space        = regexp.MustCompile(`\s+`)
	nonEntry     = regexp.MustCompile(`(?i)^(saldo|total|resumo)\b`)
	expenseHint  = regexp.MustCompile(`(?i)\b(pix enviado|pagamento|compra|d[eé]bito|saque|tarifa|boleto|imposto|transfer[eê]ncia enviada)\b`)
	incomeHint   = regexp.MustCompile(`(?i)\b(pix recebido|cr[eé]dito|sal[aá]rio|dep[oó]sito|estorno|transfer[eê]ncia recebida)\b`)
)

func ParsePDF(data []byte, bank Bank, fallbackYear int) ([]Entry, error) {
	if len(data) == 0 || len(data) > MaxStatementSize || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, domain.ErrInvalidStatement
	}
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open statement pdf: %w", domain.ErrInvalidStatement)
	}
	positioned, positionedErr := positionedText(reader)
	if positionedErr == nil {
		entries, parseErr := ParseText(positioned, bank, fallbackYear)
		if parseErr == nil {
			return entries, nil
		}
	}
	// Some PDFs do not expose usable glyph positions. Keep plain extraction as a
	// fallback for those documents.
	plain, err := reader.GetPlainText()
	if err != nil {
		return nil, fmt.Errorf("extract statement text: %w", domain.ErrInvalidStatement)
	}
	text, err := io.ReadAll(io.LimitReader(plain, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("read statement text: %w", domain.ErrInvalidStatement)
	}
	return ParseText(string(text), bank, fallbackYear)
}

func positionedText(reader *pdf.Reader) (string, error) {
	var document strings.Builder
	for pageNumber := 1; pageNumber <= reader.NumPage(); pageNumber++ {
		rows, err := pageRows(reader.Page(pageNumber))
		if err != nil {
			return "", err
		}
		for _, row := range rows {
			document.WriteString(row)
			document.WriteByte('\n')
		}
	}
	if document.Len() == 0 {
		return "", errors.New("pdf has no positioned text")
	}
	return document.String(), nil
}

func pageRows(page pdf.Page) (rows []string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("extract positioned text: %v", recovered)
		}
	}()
	texts := append([]pdf.Text(nil), page.Content().Text...)
	sort.SliceStable(texts, func(i, j int) bool {
		if texts[i].Y == texts[j].Y {
			return texts[i].X < texts[j].X
		}
		return texts[i].Y > texts[j].Y
	})
	for start := 0; start < len(texts); {
		end := start + 1
		for end < len(texts) && math.Abs(texts[end].Y-texts[start].Y) <= 1.5 {
			end++
		}
		line := append([]pdf.Text(nil), texts[start:end]...)
		sort.SliceStable(line, func(i, j int) bool { return line[i].X < line[j].X })
		var row strings.Builder
		lastEnd := 0.0
		for index, glyph := range line {
			gap := glyph.X - lastEnd
			if index > 0 && gap > math.Max(glyph.FontSize*0.22, 1.2) {
				row.WriteByte(' ')
			}
			row.WriteString(glyph.S)
			lastEnd = math.Max(lastEnd, glyph.X+glyph.W)
		}
		if value := cleanLine(row.String()); value != "" {
			rows = append(rows, value)
		}
		start = end
	}
	return rows, nil
}

func ParseText(text string, bank Bank, fallbackYear int) ([]Entry, error) {
	if !bank.Valid() {
		return nil, domain.ErrUnsupportedBank
	}
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = strings.ReplaceAll(text, "\r", "\n")
	if strings.TrimSpace(text) == "" || !utf8.ValidString(text) {
		return nil, domain.ErrStatementEmpty
	}
	year := statementYear(text, fallbackYear)
	// PDF text streams often lose visual row breaks. Dates are reliable row anchors.
	text = dateAnywhere.ReplaceAllStringFunc(text, func(value string) string { return "\n" + value })
	lines := strings.Split(text, "\n")
	entries := make([]Entry, 0)
	var pending string
	for _, raw := range lines {
		line := cleanLine(raw)
		if line == "" {
			continue
		}
		if dateAtStart.MatchString(line) {
			if pending != "" {
				if entry, ok := parseLine(pending, bank, year); ok {
					entries = append(entries, entry)
				}
			}
			pending = line
		} else if pending != "" {
			pending += " " + line
		}
		if pending != "" {
			if entry, ok := parseLine(pending, bank, year); ok {
				entries = append(entries, entry)
				pending = ""
			}
		}
	}
	if pending != "" {
		if entry, ok := parseLine(pending, bank, year); ok {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil, domain.ErrStatementEmpty
	}
	return entries, nil
}

func (bank Bank) Valid() bool {
	return bank == BankItau || bank == BankBradesco || bank == BankInter
}

func parseLine(line string, bank Bank, fallbackYear int) (Entry, bool) {
	dateMatch := dateAtStart.FindStringSubmatch(line)
	if dateMatch == nil {
		return Entry{}, false
	}
	remainder := strings.TrimSpace(line[len(dateMatch[0]):])
	amountMatch := amountAtEnd.FindStringSubmatch(remainder)
	if amountMatch == nil {
		return Entry{}, false
	}
	description := strings.TrimSpace(remainder[:len(remainder)-len(amountMatch[0])])
	description = strings.Trim(description, "-–—· |")
	if description == "" || nonEntry.MatchString(description) {
		return Entry{}, false
	}
	date, err := civilDate(dateMatch[1], dateMatch[2], dateMatch[3], fallbackYear)
	if err != nil {
		return Entry{}, false
	}
	amount, err := parseBRL(amountMatch[1])
	if err != nil || amount == 0 {
		return Entry{}, false
	}
	marker := strings.ToUpper(strings.TrimSpace(amountMatch[2]))
	kind := inferKind(bank, description, amountMatch[1], marker)
	return Entry{Kind: kind, AmountCents: amount, Description: description, Date: date}, true
}

func inferKind(_ Bank, description, rawAmount, marker string) domain.TransactionKind {
	if strings.HasPrefix(strings.TrimSpace(rawAmount), "-") || marker == "D" || marker == "-" {
		return domain.Expense
	}
	if strings.HasPrefix(strings.TrimSpace(rawAmount), "+") || marker == "C" || marker == "+" {
		return domain.Income
	}
	if expenseHint.MatchString(description) {
		return domain.Expense
	}
	if incomeHint.MatchString(description) {
		return domain.Income
	}
	// Itaú, Bradesco and Inter usually mark debits. An unmarked value is a credit.
	return domain.Income
}

func parseBRL(value string) (int64, error) {
	value = strings.ReplaceAll(value, " ", "")
	value = strings.TrimPrefix(value, "+")
	value = strings.TrimPrefix(value, "-")
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, ",", "")
	return strconv.ParseInt(value, 10, 64)
}

func civilDate(day, month, year string, fallbackYear int) (string, error) {
	if len(year) == 2 {
		year = "20" + year
	}
	if year == "" {
		year = strconv.Itoa(fallbackYear)
	}
	value := year + "-" + month + "-" + day
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Format("2006-01-02") != value {
		return "", errors.New("invalid statement date")
	}
	return value, nil
}

func statementYear(text string, fallback int) int {
	matches := fullDate.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if year, err := strconv.Atoi(match[1]); err == nil {
			return year
		}
	}
	return fallback
}

func cleanLine(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' {
			return ' '
		}
		return r
	}, value)
	return strings.TrimSpace(space.ReplaceAllString(value, " "))
}

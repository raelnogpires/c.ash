package importer

import (
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

	"c.ash/internal/domain"
	"golang.org/x/text/unicode/norm"
)

var csvAliases = map[string]map[string]struct{}{
	"date":        aliasSet("data", "date", "data transacao", "data da transacao", "data lancamento", "data do lancamento", "data movimento", "data do movimento", "data movimentacao", "transaction date", "posted date", "dtposted", "occurrence date"),
	"description": aliasSet("descricao", "description", "descricao transacao", "transaction description", "historico", "historico transacao", "history", "memo", "nome", "name", "lancamento", "detalhe", "details", "narrative"),
	"amount":      aliasSet("valor", "valor r", "amount", "amount brl", "valor transacao", "valor da transacao", "valor lancamento", "transaction amount", "transaction value", "montante", "quantia", "valor com sinal", "signed amount"),
	"type":        aliasSet("tipo", "type", "natureza", "operacao", "debito credito", "credito debito", "d c", "dc", "transaction type", "tipo transacao"),
	"debit":       aliasSet("debito", "debitos", "debit", "valor debito", "valor de debito", "debit amount", "saida", "saidas", "withdrawal", "withdrawals"),
	"credit":      aliasSet("credito", "creditos", "credit", "valor credito", "valor de credito", "credit amount", "entrada", "entradas", "deposit", "deposits"),
}

// ParseCSV infers a common statement layout from normalized column headers.
func ParseCSV(data []byte) ([]Entry, error) {
	text, err := decodeStatementText(data)
	if err != nil {
		return nil, err
	}
	delimiter, err := detectCSVDelimiter(text)
	if err != nil {
		return nil, domain.ErrInvalidStatement
	}
	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = delimiter
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, domain.ErrInvalidStatement
	}
	reader.FieldsPerRecord = len(header)
	columns, err := resolveCSVColumns(header)
	if err != nil {
		return nil, domain.ErrInvalidStatement
	}

	var rows [][]string
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, domain.ErrInvalidStatement
		}
		if recordBlank(record) {
			continue
		}
		rows = append(rows, record)
	}
	if len(rows) == 0 {
		return nil, domain.ErrStatementEmpty
	}

	signedMode := csvSignedMode(rows, columns)
	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		entry, rowErr := parseCSVRow(row, columns, signedMode)
		if rowErr != nil {
			return nil, domain.ErrInvalidStatement
		}
		entries = append(entries, entry)
	}
	if len(entries) == 0 {
		return nil, domain.ErrStatementEmpty
	}
	return entries, nil
}

type csvColumns struct {
	date, description int
	amount, kind      int
	debit, credit     int
	amountSigned      bool
}

func resolveCSVColumns(header []string) (csvColumns, error) {
	columns := csvColumns{date: -1, description: -1, amount: -1, kind: -1, debit: -1, credit: -1}
	for index, raw := range header {
		normalized := normalizeCSVLabel(raw)
		for role, aliases := range csvAliases {
			if _, found := aliases[normalized]; !found {
				continue
			}
			target := columnTarget(&columns, role)
			if *target >= 0 {
				return columns, errors.New("ambiguous CSV headers")
			}
			*target = index
			if role == "amount" && (normalized == "valor com sinal" || normalized == "signed amount") {
				columns.amountSigned = true
			}
		}
	}
	if columns.date < 0 || columns.description < 0 {
		return columns, errors.New("missing CSV headers")
	}
	split := columns.debit >= 0 || columns.credit >= 0
	if split {
		if columns.debit < 0 || columns.credit < 0 || columns.amount >= 0 {
			return columns, errors.New("incomplete CSV debit and credit columns")
		}
	} else if columns.amount < 0 {
		return columns, errors.New("missing CSV amount")
	}
	return columns, nil
}

func columnTarget(columns *csvColumns, role string) *int {
	switch role {
	case "date":
		return &columns.date
	case "description":
		return &columns.description
	case "amount":
		return &columns.amount
	case "type":
		return &columns.kind
	case "debit":
		return &columns.debit
	default:
		return &columns.credit
	}
}

func parseCSVRow(row []string, columns csvColumns, signedMode bool) (Entry, error) {
	date, err := parseCSVDate(row[columns.date])
	if err != nil {
		return Entry{}, err
	}
	description := cleanLine(row[columns.description])
	if description == "" {
		return Entry{}, errors.New("missing CSV description")
	}

	var kind domain.TransactionKind
	var cents int64
	if columns.debit >= 0 {
		debitRaw, creditRaw := strings.TrimSpace(row[columns.debit]), strings.TrimSpace(row[columns.credit])
		if isCSVAmountBlank(debitRaw) == isCSVAmountBlank(creditRaw) {
			return Entry{}, errors.New("ambiguous CSV debit and credit values")
		}
		raw := creditRaw
		kind = domain.Income
		if !isCSVAmountBlank(debitRaw) {
			raw = debitRaw
			kind = domain.Expense
		}
		value, _, parseErr := parseMoneyCents(raw)
		if parseErr != nil || value == 0 {
			return Entry{}, errors.New("invalid CSV amount")
		}
		if value < 0 {
			value = -value
		}
		cents = value
	} else {
		value, explicitSign, parseErr := parseMoneyCents(row[columns.amount])
		if parseErr != nil || value == 0 {
			return Entry{}, errors.New("invalid CSV amount")
		}
		markerKind, hasMarker := domain.TransactionKind(""), false
		if columns.kind >= 0 {
			markerKind, hasMarker = parseCSVKind(row[columns.kind])
			if !hasMarker {
				return Entry{}, errors.New("invalid CSV transaction type")
			}
		}
		if hasMarker {
			if explicitSign && ((value < 0 && markerKind != domain.Expense) || (value > 0 && markerKind != domain.Income)) {
				return Entry{}, errors.New("conflicting CSV amount and type")
			}
			kind = markerKind
		} else {
			if !explicitSign && !signedMode {
				return Entry{}, errors.New("unsigned CSV amount has no type")
			}
			kind = domain.Income
			if value < 0 {
				kind = domain.Expense
			}
		}
		if value < 0 {
			value = -value
		}
		cents = value
	}
	return Entry{Kind: kind, AmountCents: cents, Description: description, Date: date}, nil
}

func csvSignedMode(rows [][]string, columns csvColumns) bool {
	if columns.amount < 0 || columns.kind >= 0 {
		return false
	}
	if columns.amountSigned {
		return true
	}
	for _, row := range rows {
		_, explicit, err := parseMoneyCents(row[columns.amount])
		if err == nil && explicit {
			return true
		}
	}
	return false
}

func parseCSVKind(value string) (domain.TransactionKind, bool) {
	switch normalizeCSVLabel(value) {
	case "d", "debito", "debit", "despesa", "expense", "saida", "withdrawal", "-":
		return domain.Expense, true
	case "c", "credito", "credit", "receita", "income", "entrada", "deposit", "+":
		return domain.Income, true
	default:
		return "", false
	}
}

func parseCSVDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"02/01/2006", "02-01-2006", "2006-01-02", "2006/01/02"} {
		if date, err := time.Parse(layout, value); err == nil {
			return date.Format("2006-01-02"), nil
		}
	}
	return "", errors.New("invalid CSV date")
}

func parseMoneyCents(raw string) (int64, bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false, errors.New("empty amount")
	}
	negative, explicit := false, false
	if strings.HasPrefix(value, "(") && strings.HasSuffix(value, ")") {
		negative, explicit = true, true
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	value = strings.TrimSpace(strings.NewReplacer("R$", "", "r$", "", "$", "", "€", "", "£", "", "\u00a0", "", " ", "", "'", "").Replace(value))
	if strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		explicit = true
		negative = value[0] == '-'
		value = value[1:]
	}
	if value == "" {
		return 0, explicit, errors.New("missing digits")
	}
	for _, r := range value {
		if !unicode.IsDigit(r) && r != ',' && r != '.' {
			return 0, explicit, errors.New("invalid amount character")
		}
	}
	decimalAt := -1
	lastComma, lastDot := strings.LastIndex(value, ","), strings.LastIndex(value, ".")
	if lastComma >= 0 && lastDot >= 0 {
		if lastComma > lastDot {
			decimalAt = lastComma
		} else {
			decimalAt = lastDot
		}
	} else if lastComma >= 0 || lastDot >= 0 {
		separator := lastComma
		if separator < 0 {
			separator = lastDot
		}
		fractionDigits := len(value) - separator - 1
		if fractionDigits == 1 || fractionDigits == 2 {
			decimalAt = separator
		}
	}
	if err := validateAmountGrouping(value, decimalAt); err != nil {
		return 0, explicit, err
	}

	integerPart, fractionalPart := value, ""
	if decimalAt >= 0 {
		integerPart, fractionalPart = value[:decimalAt], value[decimalAt+1:]
	}
	integerPart = strings.NewReplacer(",", "", ".", "").Replace(integerPart)
	if integerPart == "" || strings.ContainsAny(fractionalPart, ",.") || len(fractionalPart) > 2 {
		return 0, explicit, errors.New("invalid amount notation")
	}
	if len(fractionalPart) == 1 {
		fractionalPart += "0"
	}
	if fractionalPart == "" {
		fractionalPart = "00"
	}
	units, err := strconv.ParseInt(integerPart, 10, 64)
	if err != nil {
		return 0, explicit, err
	}
	fraction, err := strconv.ParseInt(fractionalPart, 10, 64)
	if err != nil {
		return 0, explicit, err
	}
	if units > (1<<63-1-fraction)/100 {
		return 0, explicit, errors.New("amount overflow")
	}
	cents := units*100 + fraction
	if negative {
		cents = -cents
	}
	return cents, explicit, nil
}

func validateAmountGrouping(value string, decimalAt int) error {
	integerPart := value
	decimalSeparator := byte(0)
	if decimalAt >= 0 {
		integerPart = value[:decimalAt]
		decimalSeparator = value[decimalAt]
		if strings.ContainsRune(integerPart, rune(decimalSeparator)) {
			return errors.New("repeated decimal separator")
		}
	}
	separator := byte(0)
	if strings.Contains(integerPart, ",") {
		separator = ','
	}
	if strings.Contains(integerPart, ".") {
		if separator != 0 {
			return errors.New("mixed thousands separators")
		}
		separator = '.'
	}
	if separator == 0 {
		return nil
	}
	if separator == decimalSeparator {
		return errors.New("decimal separator in integer part")
	}
	groups := strings.Split(integerPart, string(separator))
	if len(groups[0]) < 1 || len(groups[0]) > 3 {
		return errors.New("invalid leading amount group")
	}
	for _, group := range groups[1:] {
		if len(group) != 3 {
			return errors.New("invalid thousands group")
		}
	}
	return nil
}

func detectCSVDelimiter(text string) (rune, error) {
	bestDelimiter, bestFields := rune(0), 1
	for _, delimiter := range []rune{',', ';', '\t'} {
		reader := csv.NewReader(strings.NewReader(text))
		reader.Comma = delimiter
		record, err := reader.Read()
		if err == nil && len(record) > bestFields {
			bestDelimiter, bestFields = delimiter, len(record)
		}
	}
	if bestDelimiter == 0 {
		return 0, errors.New("CSV delimiter not found")
	}
	return bestDelimiter, nil
}

func normalizeCSVLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = norm.NFD.String(value)
	value = strings.Map(func(r rune) rune {
		switch {
		case unicode.Is(unicode.Mn, r):
			return -1
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' || r == '-':
			return r
		default:
			return ' '
		}
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func aliasSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[normalizeCSVLabel(value)] = struct{}{}
	}
	return set
}

func recordBlank(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func isCSVAmountBlank(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return true
	}
	cents, _, err := parseMoneyCents(value)
	return err == nil && cents == 0
}

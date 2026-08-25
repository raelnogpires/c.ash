package importer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"c.ash/internal/domain"
	"github.com/ledongthuc/pdf"
)

func TestParseText_SupportedBankLayouts(t *testing.T) {
	tests := []struct {
		name string
		bank Bank
		text string
	}{
		{"itau", BankItau, "Itaú\n01/08/2026 PIX RECEBIDO MARIA 1.250,00 C\n02/08/2026 COMPRA MERCADO 89,90 D"},
		{"bradesco", BankBradesco, "Bradesco\n03/08/2026 PAGAMENTO BOLETO\nR$ 140,25 D\n04/08/2026 DEPÓSITO DINHEIRO 200,00 C"},
		{"inter", BankInter, "Banco Inter\n05/08/2026 Pix enviado - JOAO -50,00\n06/08/2026 Pix recebido - ANA +75,50"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := ParseText(tc.text, tc.bank, 2026)
			if err != nil || len(entries) != 2 {
				t.Fatalf("entries=%+v err=%v", entries, err)
			}
			if entries[0].Kind == entries[1].Kind || entries[0].AmountCents == 0 || entries[0].Date[:4] != "2026" {
				t.Fatalf("unexpected entries: %+v", entries)
			}
		})
	}
}

func TestParseOFX_SupportsSGMLAndXML(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "OFX 1 SGML",
			data: "OFXHEADER:100\nDATA:OFXSGML\nVERSION:102\n\n<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS><BANKTRANLIST>" +
				"<STMTTRN><TRNTYPE>DEBIT<DTPOSTED>20260802120000[-3:BRT]<TRNAMT>-25.90<NAME>PIX JOAO<MEMO>Almoço</STMTTRN>" +
				"<STMTTRN><TRNTYPE>CREDIT<DTPOSTED>20260805<TRNAMT>1000.00<NAME>Salário</STMTTRN>" +
				"</BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>",
		},
		{
			name: "OFX 2 XML",
			data: `OFXHEADER:200
DATA:OFXSGML
VERSION:202

<?xml version="1.0" encoding="UTF-8"?>
<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS><BANKTRANLIST>
<STMTTRN><DTPOSTED>20260802120000.000[-3:BRT]</DTPOSTED><TRNAMT>-25.90</TRNAMT><NAME>PIX JOAO</NAME><MEMO>Almoço</MEMO></STMTTRN>
<STMTTRN><DTPOSTED>20260805</DTPOSTED><TRNAMT>1000.00</TRNAMT><NAME>Salário</NAME></STMTTRN>
</BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := ParseOFX([]byte(tc.data))
			if err != nil || len(entries) != 2 {
				t.Fatalf("entries=%+v err=%v", entries, err)
			}
			if entries[0].Kind != domain.Expense || entries[0].AmountCents != 2590 || entries[0].Date != "2026-08-02" || entries[0].Description != "PIX JOAO" {
				t.Fatalf("expense=%+v", entries[0])
			}
			if entries[1].Kind != domain.Income || entries[1].AmountCents != 100000 {
				t.Fatalf("income=%+v", entries[1])
			}
		})
	}
}

func TestParseOFX_DecodesWindows1252AndRejectsMalformedDocuments(t *testing.T) {
	data := []byte("OFXHEADER:100\nDATA:OFXSGML\nVERSION:102\n<OFX><BANKTRANLIST><STMTTRN><DTPOSTED>20260801<TRNAMT>-10.00<MEMO>Caf")
	data = append(data, 0xe9)
	data = append(data, []byte("</STMTTRN></BANKTRANLIST></OFX>")...)
	entries, err := ParseOFX(data)
	if err != nil || len(entries) != 1 || entries[0].Description != "Café" {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
	for _, malformed := range []string{
		"not OFX",
		"<OFX><STMTTRN><DTPOSTED>20260801<TRNAMT>-10.00<NAME>broken</OFX>",
		`<?xml version="1.0"?><OFX><STMTTRN><DTPOSTED>20260801</DTPOSTED><TRNAMT>-1.00</TRNAMT></OFX>`,
	} {
		if _, err := ParseOFX([]byte(malformed)); !errors.Is(err, domain.ErrInvalidStatement) {
			t.Fatalf("malformed=%q error=%v", malformed, err)
		}
	}
	if _, err := ParseOFX([]byte("OFXHEADER:100\n<OFX><BANKTRANLIST></BANKTRANLIST></OFX>")); !errors.Is(err, domain.ErrStatementEmpty) {
		t.Fatalf("empty error=%v", err)
	}
}

func TestParseCSV_InfersSupportedLayoutsAndNumberFormats(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want []Entry
	}{
		{
			name: "semicolon signed BRL with BOM",
			data: []byte("\xef\xbb\xbfData;Descrição;Valor\r\n01/08/2026;Salário;+1.234,56\r\n02/08/2026;Mercado;-25,90\r\n"),
			want: []Entry{{Kind: domain.Income, AmountCents: 123456, Description: "Salário", Date: "2026-08-01"}, {Kind: domain.Expense, AmountCents: 2590, Description: "Mercado", Date: "2026-08-02"}},
		},
		{
			name: "comma signed international",
			data: []byte("date,description,amount\n2026-08-01,Salary,\"+1,234.56\"\n2026-08-02,Market,-25.90\n"),
			want: []Entry{{Kind: domain.Income, AmountCents: 123456, Description: "Salary", Date: "2026-08-01"}, {Kind: domain.Expense, AmountCents: 2590, Description: "Market", Date: "2026-08-02"}},
		},
		{
			name: "tab with debit credit marker",
			data: []byte("Data da transação\tHistórico\tQuantia\tD/C\n01/08/2026\tPagamento\t25,90\tD\n02/08/2026\tDepósito\t100,00\tC\n"),
			want: []Entry{{Kind: domain.Expense, AmountCents: 2590, Description: "Pagamento", Date: "2026-08-01"}, {Kind: domain.Income, AmountCents: 10000, Description: "Depósito", Date: "2026-08-02"}},
		},
		{
			name: "split debit and credit",
			data: []byte("Data;Lançamento;Débito;Crédito\n01/08/2026;Conta de luz;125,00;0,00\n2026-08-02;Estorno;;10,50\n"),
			want: []Entry{{Kind: domain.Expense, AmountCents: 12500, Description: "Conta de luz", Date: "2026-08-01"}, {Kind: domain.Income, AmountCents: 1050, Description: "Estorno", Date: "2026-08-02"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := ParseCSV(tc.data)
			if err != nil || len(entries) != len(tc.want) {
				t.Fatalf("entries=%+v err=%v", entries, err)
			}
			for index := range tc.want {
				if entries[index] != tc.want[index] {
					t.Fatalf("entry[%d]=%+v want=%+v", index, entries[index], tc.want[index])
				}
			}
		})
	}
}

func TestParseCSV_DecodesWindows1252(t *testing.T) {
	data := []byte("Data;Descri")
	data = append(data, 0xe7, 0xe3)
	data = append(data, []byte("o;Valor\n01/08/2026;Caf")...)
	data = append(data, 0xe9)
	data = append(data, []byte(";-5,50\n")...)
	entries, err := ParseCSV(data)
	if err != nil || len(entries) != 1 || entries[0].Description != "Café" {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
}

func TestParseCSV_RejectsMissingMalformedAndAmbiguousContent(t *testing.T) {
	tests := []struct {
		name string
		data string
		err  error
	}{
		{"missing headers", "Data;Descrição\n01/08/2026;Teste\n", domain.ErrInvalidStatement},
		{"ambiguous unsigned amount", "Data;Descrição;Valor\n01/08/2026;Teste;10,00\n", domain.ErrInvalidStatement},
		{"malformed date", "Data;Descrição;Valor\n31/02/2026;Teste;-10,00\n", domain.ErrInvalidStatement},
		{"malformed row", "Data;Descrição;Valor\n01/08/2026;Teste;-10,00;extra\n", domain.ErrInvalidStatement},
		{"malformed amount grouping", "Data;Descrição;Valor\n01/08/2026;Teste;-12,34,567\n", domain.ErrInvalidStatement},
		{"empty", "Data;Descrição;Valor\n", domain.ErrStatementEmpty},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseCSV([]byte(tc.data)); !errors.Is(err, tc.err) {
				t.Fatalf("error=%v want=%v", err, tc.err)
			}
		})
	}
}

func TestParsers_EnforceSharedSizeLimit(t *testing.T) {
	tooLarge := []byte(strings.Repeat("x", MaxStatementSize+1))
	if _, err := ParseOFX(tooLarge); !errors.Is(err, domain.ErrInvalidStatement) {
		t.Fatalf("OFX size error=%v", err)
	}
	if _, err := ParseCSV(tooLarge); !errors.Is(err, domain.ErrInvalidStatement) {
		t.Fatalf("CSV size error=%v", err)
	}
}

func TestParsePDF_ExtractsTextWithoutPersistingTheFile(t *testing.T) {
	content := "BT /F1 12 Tf 72 720 Td (Itau) Tj 0 -20 Td (01/08/2026 PIX TRANSF RAEL NO01/08 100,00 C) Tj 0 -20 Td (02/08/2026 COMPRA MERCADO 25,50 D) Tj 0 -20 Td (03/08/2026 JUROS LIMITE DA CONTA -1,90) Tj ET"
	data := testPDF(content)
	entries, err := ParsePDF(data, BankItau, 2026)
	if err != nil || len(entries) != 3 || entries[0].Kind != domain.Income || entries[1].Kind != domain.Expense || entries[2].Description != "JUROS LIMITE DA CONTA" || entries[0].Date != "2026-08-01" {
		reader, _ := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
		plain, _ := reader.GetPlainText()
		extracted, _ := io.ReadAll(plain)
		t.Logf("extracted=%q", extracted)
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
}

func testPDF(content string) []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		"<< /Length " + strconv.Itoa(len(content)) + " >>\nstream\n" + content + "\nendstream",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for index := 1; index <= len(objects); index++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return pdf.Bytes()
}

func TestParseText_RejectsUnsupportedOrEmptyStatements(t *testing.T) {
	if _, err := ParseText("01/08/2026 PIX 10,00 C", "other", 2026); !errors.Is(err, domain.ErrUnsupportedBank) {
		t.Fatalf("unsupported error=%v", err)
	}
	if _, err := ParseText("Itaú - somente saldo", BankItau, 2026); !errors.Is(err, domain.ErrStatementEmpty) {
		t.Fatalf("empty error=%v", err)
	}
}

package importer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
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

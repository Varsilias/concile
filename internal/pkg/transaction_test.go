package pkg

import "testing"

func BenchmarkNormalizeInflow(b *testing.B) {
	raw := RawTransaction{
		Reference:     "Zpay-20220217064012271",
		FromAccountNo: "4243914279",
		FromBank:      "Access Bank",
		ToAccountNo:   "5773153770",
		SessionID:     "941848771823095733170191134194",
		Date:          "2022-02-17 06:40:12",
		Amount:        "3,464,883.32",
		Type:          "INFLOW",
		Wallet:        "Zpay",
	}

	for b.Loop() {
		_, _ = Normalize(raw)
	}
}

func BenchmarkNormalizeOutflow(b *testing.B) {
	raw := RawTransaction{
		Reference:     "v1-zpay-959dc989-8ca4-4531-8ee3-76df032627b1",
		SessionID:     "414002479498026811172690138178",
		Date:          "2022-04-02 01:42:46",
		OutflowAmount: "24000",
		Type:          "OUTFLOW",
		StatementID:   "31356989-31356993",
		ResponseCode:  "00",
	}

	for b.Loop() {
		_, _ = Normalize(raw)
	}
}

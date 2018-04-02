package spvwallet

import (
	"bytes"
	"github.com/OpenBazaar/wallet-interface"
	"net/http"
	"testing"
)

type ClosingBuffer struct {
	*bytes.Buffer
}

func (cb *ClosingBuffer) Close() (err error) {
	return
}

type mockHttpClient struct{}

func (m *mockHttpClient) Get(url string) (*http.Response, error) {
	data := `{"priority":450,"normal":420,"economic":390}`
	cb := &ClosingBuffer{bytes.NewBufferString(data)}
	resp := &http.Response{
		Body: cb,
	}
	return resp, nil
}

func TestFeeProvider_GetFeePerByte(t *testing.T) {
	fp := NewFeeProvider(2000, 360, 320, 280, "https://btc.fees.openbazaar.org", nil)
	fp.httpClient = new(mockHttpClient)

	// Test fetch from API
	if fp.GetFeePerByte(wallet.PRIOIRTY) != 450 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.NORMAL) != 420 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.ECONOMIC) != 390 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.FEE_BUMP) != 450 {
		t.Error("Returned incorrect fee per byte")
	}

	// Test return over max
	fp.maxFee = 100
	if fp.GetFeePerByte(wallet.PRIOIRTY) != 100 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.NORMAL) != 100 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.ECONOMIC) != 100 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.FEE_BUMP) != 100 {
		t.Error("Returned incorrect fee per byte")
	}

	// Test no API provided
	fp.feeAPI = ""
	if fp.GetFeePerByte(wallet.PRIOIRTY) != 360 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.NORMAL) != 320 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.ECONOMIC) != 280 {
		t.Error("Returned incorrect fee per byte")
	}
	if fp.GetFeePerByte(wallet.FEE_BUMP) != 360 {
		t.Error("Returned incorrect fee per byte")
	}
}

package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ecadlabs/signatory/pkg/server/auth"
	"github.com/ecadlabs/signatory/pkg/signatory"
	"github.com/stretchr/testify/require"
)

type fakeSignatory struct {
	SignResponse      string
	SignError         error
	PublicKeyResponse *signatory.PublicKey
	PublicKeyError    error
}

func (c *fakeSignatory) Sign(ctx context.Context, keyHash string, message []byte) (string, error) {
	return c.SignResponse, c.SignError
}

func (c *fakeSignatory) GetPublicKey(ctx context.Context, keyHash string) (*signatory.PublicKey, error) {
	return c.PublicKeyResponse, c.PublicKeyError
}

func toReadCloser(str string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader([]byte(str)))
}

func TestSign(t *testing.T) {
	type testCase struct {
		Name       string
		Request    string
		StatusCode int
		Response   string
		Error      error
		Expected   string
	}

	cases := []testCase{
		{
			Name:       "Bad request",
			StatusCode: http.StatusBadRequest,
			Expected:   "[{\"id\":\"failure\",\"kind\":\"temporary\",\"msg\":\"unexpected end of JSON input\"}]\n",
		},
		{
			Name:       "Invalid body",
			Request:    "\x03",
			StatusCode: http.StatusBadRequest,
			Expected:   "[{\"id\":\"failure\",\"kind\":\"temporary\",\"msg\":\"invalid character '\\\\x03' looking for beginning of value\"}]\n",
		},
		{
			Name:       "Invalid hex",
			Request:    "\"03ZZZZ\"",
			StatusCode: http.StatusBadRequest,
			Expected:   "[{\"id\":\"failure\",\"kind\":\"temporary\",\"msg\":\"encoding/hex: invalid byte: U+005A 'Z'\"}]\n",
		},
		{
			Name:       "Ok",
			Request:    "\"03123453\"",
			StatusCode: http.StatusOK,
			Response:   "signature",
			Expected:   "{\"signature\":\"signature\"}\n",
		},
		{
			Name:       "Signature error",
			Request:    "\"03123453\"",
			StatusCode: http.StatusInternalServerError,
			Error:      errors.New("error"),
			Expected:   "[{\"id\":\"failure\",\"kind\":\"temporary\",\"msg\":\"error\"}]\n",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			sig := &fakeSignatory{
				SignError:    c.Error,
				SignResponse: c.Response,
			}

			srv := &Server{
				Signer: sig,
			}

			var body io.Reader
			if c.Request != "" {
				body = strings.NewReader(c.Request)
			}

			r := httptest.NewRequest("POST", "http://irrelevant.com/keys/03123453", body)
			resp := httptest.NewRecorder()
			srv.New().Handler.ServeHTTP(resp, r)

			require.Equal(t, c.StatusCode, resp.Code)

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf(err.Error())
			}

			require.Equal(t, c.Expected, string(b))
		})
	}
}

func TestGetPublicKey(t *testing.T) {
	type testCase struct {
		Name       string
		StatusCode int
		Response   *signatory.PublicKey
		Error      error
		Expected   string
	}

	cases := []testCase{
		{
			Name:       "Read Error",
			StatusCode: http.StatusInternalServerError,
			Error:      errors.New("test"),
			Expected:   "[{\"id\":\"failure\",\"kind\":\"temporary\",\"msg\":\"test\"}]\n",
		},
		{
			Name:       "Normal case",
			StatusCode: http.StatusOK,
			Response:   &signatory.PublicKey{PublicKey: "key"},
			Expected:   "{\"public_key\":\"key\"}\n",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			sig := &fakeSignatory{
				PublicKeyError:    c.Error,
				PublicKeyResponse: c.Response,
			}

			srv := &Server{
				Signer: sig,
			}

			r := httptest.NewRequest("GET", "http://irrelevant.com/keys/03123453", nil)
			resp := httptest.NewRecorder()
			srv.New().Handler.ServeHTTP(resp, r)

			require.Equal(t, c.StatusCode, resp.Code)

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf(err.Error())
			}

			require.Equal(t, c.Expected, string(b))
		})
	}
}

func TestSignedRequest(t *testing.T) {
	type testCase struct {
		Name       string
		Signature  string
		StatusCode int
	}

	cases := []testCase{
		{
			Name:       "Ok",
			Signature:  "edsigu1n7Zw1mvwmM22attD7Jwoy3MXFXJU3WAqQeww2RuRr1kxhEjEvkW9L1wD7h1EnHaMuqFWJ6qkAGuW4enmq8CdRSw45k5W",
			StatusCode: http.StatusOK,
		},
		{
			Name:       "Unauthorized",
			StatusCode: http.StatusUnauthorized,
		},
		{
			Name:       "Forbidden",
			Signature:  "spsig1SbAZ2AWQP6fXYusCW8XowTxieZw874YcuBtKYkGEEDrvyTgReLY3jKAuoBamBALRtrEsEMG5N7zxmuxfE9MDLgsMP1YJh",
			StatusCode: http.StatusForbidden,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			sig := &fakeSignatory{
				SignResponse: "signature",
			}

			srv := &Server{
				Signer: sig,
				Auth:   auth.Must(auth.StaticAuthorizedKeysFromString("edpktpQKJF4vRodmSfT3h6LrYisshQuJeoybUxB9c8s3b1QymvisHC")),
			}

			body := strings.NewReader("\"03a11f5f176e553a11cf184bb2b15f09f55dfc5dcb2d26d79bf5dd099d074d5f5d6c0079cae4c9a1885f17d3995619bf28636c4394458b820af19172c35000904e0000712c4c4270d9e7f512115310d8ec6acfcd878bef00\"")
			u, _ := url.Parse("http://irrelevant.com/keys/tz1Wk1Wdczh5BzyZ1uz2DW9xdFg9B5cFuGFm")

			if c.Signature != "" {
				u.RawQuery = url.Values{
					"authentication": []string{c.Signature},
				}.Encode()
			}

			r := httptest.NewRequest("POST", u.String(), body)

			resp := httptest.NewRecorder()
			srv.New().Handler.ServeHTTP(resp, r)

			require.Equal(t, c.StatusCode, resp.Code)
		})
	}
}

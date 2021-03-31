package larkbot

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/caiguanhao/larkslim"
)

type (
	Server struct {
		GetAccessToken         func() (int, error)
		CardCallbackHandler    func(http.ResponseWriter, interface{})
		EventCallbackHandler   func(larkslim.EventResponse)
		EventEncrytionKey      string
		EventVerificationToken string

		Logger interface {
			Debug(args ...interface{})
			Info(args ...interface{})
			Error(args ...interface{})
			Fatal(args ...interface{})
		}
	}
)

func (h *Server) Serve(address string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cards/", h.handleLarkCards)
	mux.HandleFunc("/events/", h.handleLarkEvents)
	mux.HandleFunc("/204/", h.handle204)
	mux.HandleFunc("/", h.handle404)
	server := &http.Server{
		Addr:    address,
		Handler: h.logRequest(mux),
	}
	go h.updateAccessToken()
	if h.Logger != nil {
		h.Logger.Info("listening", address)
		h.Logger.Fatal(server.ListenAndServe())
	}
}

func (h *Server) updateAccessToken() {
	if h.GetAccessToken == nil {
		return
	}
	defer func() {
		time.Sleep(5 * time.Second)
		h.updateAccessToken()
	}()
	expire, err := h.GetAccessToken()
	if err != nil {
		if h.Logger != nil {
			h.Logger.Error(err)
		}
		return
	}
	secs := expire - 60
	if secs < 5 {
		secs = 5
	}
	if h.Logger != nil {
		h.Logger.Info("update access token in", secs, "seconds")
	}
	time.Sleep(time.Duration(secs) * time.Second)
}

func (h *Server) handleLarkCards(w http.ResponseWriter, r *http.Request) {
	returnError := func(err error) {
		if h.Logger != nil {
			h.Logger.Error(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		returnError(err)
		return
	}

	if h.EventVerificationToken != "" {
		var b strings.Builder
		b.WriteString(r.Header.Get("X-Lark-Request-Timestamp"))
		b.WriteString(r.Header.Get("X-Lark-Request-Nonce"))
		b.WriteString(h.EventVerificationToken)
		b.Write(body)
		bs := []byte(b.String())
		h := sha1.New()
		h.Write(bs)
		bs = h.Sum(nil)
		sig := fmt.Sprintf("%x", bs)
		fmt.Println(r.Header, sig)
		if r.Header.Get("X-Lark-Signature") != sig {
			returnError(errors.New("wrong signature"))
		}
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		returnError(err)
		return
	}
	if h.Logger != nil {
		h.Logger.Debug(string(body))
	}
	if v, ok := resp["type"]; ok {
		value, ok := v.(string)
		if !ok {
			goto end
		}
		if value != "url_verification" {
			goto end
		}
		if data, err := json.Marshal(map[string]interface{}{
			"challenge": resp["challenge"],
		}); err == nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, string(data))
			return
		}
	}
	if v, ok := resp["action"]; ok {
		if h.CardCallbackHandler != nil {
			h.CardCallbackHandler(w, v)
			return
		}
	}
end:
	w.WriteHeader(http.StatusOK)
}

func (h *Server) handleLarkEvents(w http.ResponseWriter, r *http.Request) {
	returnError := func(err error) {
		if h.Logger != nil {
			h.Logger.Error(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		returnError(err)
		return
	}

	if h.EventEncrytionKey != "" {
		key := sha256.Sum256([]byte(h.EventEncrytionKey))
		block, err := aes.NewCipher(key[:])
		if err != nil {
			returnError(err)
			return
		}
		var resp map[string]string
		if err := json.Unmarshal(body, &resp); err != nil {
			returnError(err)
			return
		}
		cipherText, err := base64.StdEncoding.DecodeString(resp["encrypt"])
		if err != nil {
			returnError(err)
			return
		}
		iv := cipherText[:aes.BlockSize]
		cipherText = cipherText[aes.BlockSize:]
		cipher.NewCBCDecrypter(block, iv).CryptBlocks(cipherText, cipherText)
		bufLen := len(cipherText) - int(cipherText[len(cipherText)-1])
		body = cipherText[:bufLen] // unpad
	}

	var resp larkslim.EventResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		returnError(err)
		return
	}
	if h.Logger != nil {
		h.Logger.Debug(string(body))
	}
	if h.EventVerificationToken != "" && h.EventVerificationToken != resp.Token {
		returnError(errors.New("wrong verification token"))
		return
	}
	switch resp.Type {
	case "url_verification":
		if data, err := json.Marshal(map[string]string{
			"challenge": resp.Challenge,
		}); err == nil {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, string(data))
			return
		}
	case "event_callback":
		if h.EventCallbackHandler != nil {
			h.EventCallbackHandler(resp)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Server) handle204(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (h *Server) handle404(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (h *Server) logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.Logger != nil {
			h.Logger.Debug(r.Method, r.URL)
		}
		handler.ServeHTTP(w, r)
	})
}

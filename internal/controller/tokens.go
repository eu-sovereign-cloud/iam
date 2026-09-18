package controller

import "net/http"

type exchangeRequest struct {
	PAT string `json:"pat"`
}

type exchangeResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (c *Controller) handleExchangeToken(w http.ResponseWriter, r *http.Request) {
	var req exchangeRequest
	if err := decodeJSON(r, &req); err != nil || req.PAT == "" {
		writeError(w, http.StatusBadRequest, "body must be JSON {\"pat\": \"...\"}")
		return
	}

	issued, err := c.Tokens.Exchange(r.Context(), req.PAT)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, exchangeResponse{
		AccessToken: issued.AccessToken,
		TokenType:   issued.TokenType,
		ExpiresIn:   issued.ExpiresIn,
	})
}

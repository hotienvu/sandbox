package main

import (
	"net/http"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/errors"
	"github.com/didip/tollbooth/libstring"
	"github.com/didip/tollbooth/limiter"
	"github.com/julienschmidt/httprouter"
	"github.com/urfave/negroni"
)

// CmdLimiter limits API calls based on the command used. If no limiter
// was specified for a given command, then a global fallback will be used.
type CmdLimiter struct {
	limiters map[string]*limiter.Limiter
	fallback *limiter.Limiter
}

func NewCmdLimiter(limits map[string]int64, def int64) *CmdLimiter {
	limiters := map[string]*limiter.Limiter{}
	for k, v := range limits {
		limiters[k] = tollbooth.NewLimiter(v, 1*time.Minute, nil)
	}

	defLim := tollbooth.NewLimiter(def, 1*time.Minute, nil)
	clim := &CmdLimiter{fallback: defLim, limiters: limiters}

	return clim
}

func (c *CmdLimiter) Limit(cmd string, r *http.Request) *errors.HTTPError {
	l, ok := c.limiters[cmd]
	remoteIP := libstring.RemoteIP(c.fallback.GetIPLookups(), 0, r)
	keys := []string{remoteIP, cmd}
	if !ok { // Use fallback if cmd was not found.
		return tollbooth.LimitByKeys(c.fallback, keys)
	}

	return tollbooth.LimitByKeys(l, keys)
}

// From: https://github.com/didip/tollbooth/tree/master/thirdparty/tollbooth_negroni

func LimitHandler(limiter *limiter.Limiter) negroni.HandlerFunc {
	return negroni.HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		httpError := tollbooth.LimitByRequest(limiter, w, r)
		if httpError != nil {
			writeError(w, httpError.StatusCode, ErrorResp{Message: httpError.Message})
			return
		} else {
			next(w, r)
		}
	})
}

func AttachLimitHandler(handler httprouter.Handle, limiter *limiter.Limiter) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		httpError := tollbooth.LimitByRequest(limiter, w, r)
		if httpError != nil {
			writeError(w, httpError.StatusCode, ErrorResp{Message: httpError.Message})
			return
		}
		handler(w, r, ps)
	}
}

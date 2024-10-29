package resp

import "net/http"

func (r *Responder) Success() error {
	if r.Message == "" {
		r.Message = "Success"
	}
	return r.e.JSON(http.StatusOK, r.Standard(http.StatusOK, http.StatusText(http.StatusOK), r.Message, r.Data))
}

func (r *Responder) RenderSuccess() error {
	return r.Success()
}

func (r *Responder) RenderCreated(insertedId string) error {
	r.Data = insertedId
	return r.Success()
}

func (r *Responder) RenderSuccessPayload(payload interface{}) error {
	r.Data = payload
	return r.Success()
}

func (r *Responder) RenderSuccessWithMessage(message string, payload interface{}) error {
	r.Message = message
	r.Data = payload
	return r.Success()
}

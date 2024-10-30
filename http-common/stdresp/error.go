package stdresp

import (
	"errors"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/venturoid/go/http-common/derror"
)

type Responder struct {
	e echo.Context
	StandardResponse
}

// StandardResponse unified response payload
type StandardResponse struct {
	Code      int         `json:"code"`
	Status    string      `json:"status,omitempty"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Signature string      `json:"data,omitempty"`
}

func New(e echo.Context) *Responder {
	return &Responder{e: e}
}

// Standard creates a response with code, status, message, and data
func (r *Responder) Standard(code int, status, message string, data interface{}) StandardResponse {
	r.StandardResponse = StandardResponse{
		Code:    code,
		Status:  status,
		Message: message,
		Data:    data,
	}
	return r.StandardResponse
}

// Error creates an error response with optional internal error logging
func (r *Responder) Error(code int, status, message string, internalError error) StandardResponse {
	if internalError != nil {
		log.Println("Internal Error", internalError.Error())
	}
	return r.Standard(code, status, message, nil)
}

// RenderErrorResponse handles rendering of different error cases
func (r *Responder) DError(err error) StandardResponse {
	// Error cases mapping
	errorCases := map[derror.ErrorCode]StandardResponse{
		derror.ErrorCodeNotFound:             r.Error(http.StatusNotFound, NotFoundStatus, "Data Not Found", nil),
		derror.ErrorCodeInvalidArgument:      r.Error(http.StatusBadRequest, BadRequestStatus, "Bad Request", nil),
		derror.ErrorCodeDuplicate:            r.Error(http.StatusBadRequest, DuplicateStatus, "Duplicate", nil),
		derror.ErrorCodeCustomBadRequest:     r.Error(http.StatusBadRequest, BadRequestStatus, "Custom Bad Request", nil),
		derror.ErrorCodeCustomNotFound:       r.Error(http.StatusNotFound, NotFoundStatus, "Custom Not Found", nil),
		derror.ErrorCodeForbidden:            r.Error(http.StatusForbidden, ForbiddenStatus, "Forbidden", nil),
		derror.ErrorRequestTimeout:           r.Error(http.StatusRequestTimeout, TimeoutStatus, "Request Timeout", nil),
		derror.ErrorUnauthorized:             r.Error(http.StatusUnauthorized, UnauthorizedStatus, "Unauthorized", nil),
		derror.ErrorCodeCustomInternalServer: r.Error(http.StatusInternalServerError, InternalErrStatus, "Internal Server Error", nil),
		derror.ErrorCodeBAG:                  r.Error(http.StatusInternalServerError, InternalErrStatus, "BAG Error", nil),
	}

	var ierr *derror.Error
	if !errors.As(err, &ierr) {
		return r.Error(http.StatusInternalServerError, InternalErrStatus, "Internal Server Error", err)
	}

	if resp, exists := errorCases[ierr.Code()]; exists {
		return resp
	}
	// Return default internal server error if no match found
	return r.Error(http.StatusInternalServerError, InternalErrStatus, "Internal Server Error", ierr)
}

// Render functions for common errors
func (r *Responder) RenderError() error {
	// return Responder directly, because e is not exported, so it will not be encoded by JSON
	return r.e.JSON(r.Code, r)
}

func (r *Responder) RenderBadRequest(data interface{}) error {
	r.Data = data
	r.Code = http.StatusBadRequest
	r.Message = http.StatusText(http.StatusBadRequest)
	return r.RenderError()
}

func (r *Responder) RenderValidationBadRequest(message string) error {
	r.Message = message
	r.Code = http.StatusBadRequest
	return r.RenderError()
}

func (r *Responder) RenderNotFound() error {
	r.Code = http.StatusNotFound
	r.Message = http.StatusText(http.StatusNotFound)
	return r.RenderError()
}

func (r *Responder) RenderForbidden() error {
	r.Code = http.StatusForbidden
	r.Message = http.StatusText(http.StatusForbidden)
	return r.RenderError()
}

func (r *Responder) RenderUnauthorized() error {
	r.Code = http.StatusUnauthorized
	r.Message = http.StatusText(http.StatusUnauthorized)
	return r.RenderError()
}

func (r *Responder) RenderInternalServerError() error {
	r.Code = http.StatusInternalServerError
	r.Message = http.StatusText(http.StatusInternalServerError)
	return r.RenderError()
}

/* This function is used to accomodate legacy code that returns module's code, message, and data directly without checking
* eg:
* response.StatusCode = cd
* response.Message = msg
* response.Data = data
* return e.JSON(response.StatusCode, response)
*
* and replacing it with this will be
* response.RenderEvaluate(cd, msg, data)
 */
func (r *Responder) RenderEvaluate(code int, message string, data interface{}) error {
	if code == http.StatusOK {
		return r.RenderSuccessWithMessage(message, data)
	}
	r.Code = code
	r.Message = message
	r.Data = data
	return r.RenderError()
}

package bucket

import (
	"net/http"
	"s3/internal/server/utils"
)

func (h *Handler) multiDeleteHandler(w http.ResponseWriter, r *http.Request) {
	reqID, _ := r.Context().Value("requestID").(string)

	utils.S3ErrorResponse(w, utils.S3Error{
		Code:      "NotImplemented",
		Message:   "A header that you provided implies functionality that is not implemented",
		RequestId: reqID,
		Resource:  r.URL.Path,
	})
}

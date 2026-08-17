package handler

import (
	"crypto/rand"
	"log/slog"
	"math/big"
	"net/http"

	"github.com/Joon-paxn/Jiuin/backend/internal/api/response"
)

var backgroundURLAllowlist = [...]string{
	"https://image1.cn-nb1.rains3.com/pc/img1.jpg",
	"https://image1.cn-nb1.rains3.com/pc/img2.jpg",
	"https://image1.cn-nb1.rains3.com/pc/img3.jpg",
	"https://image1.cn-nb1.rains3.com/pc/img4.jpg",
	"https://image1.cn-nb1.rains3.com/pc/img5.jpg",
	"https://image1.cn-nb1.rains3.com/pc/img6.jpg",	
	"https://image1.cn-nb1.rains3.com/pc/img7.jpg",
	"https://image1.cn-nb1.rains3.com/pc/img8.jpg",
	"https://image1.cn-nb1.rains3.com/pc/img9.jpg",
	"https://image1.cn-nb1.rains3.com/pc/img10.jpg",
}

type BackgroundHandler struct {
	logger *slog.Logger
}

func NewBackgroundHandler(logger *slog.Logger) BackgroundHandler {
	return BackgroundHandler{logger: logger}
}

// Random returns one URL from the server-owned allowlist. The server never
// proxies the image and accepts no caller-provided URL, keeping this endpoint
// free from arbitrary outbound requests.
func (handler BackgroundHandler) Random(w http.ResponseWriter, _ *http.Request) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(backgroundURLAllowlist))))
	if err != nil {
		handler.logger.Error("failed to choose random background", "error", err)
		response.Error(w, http.StatusInternalServerError, "failed to select background")
		return
	}

	response.Success(w, struct {
		URL string `json:"url"`
	}{URL: backgroundURLAllowlist[index.Int64()]})
}

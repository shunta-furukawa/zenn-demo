package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

var oauthConfig *oauth2.Config

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	auth0Domain := os.Getenv("AUTH0_DOMAIN")

	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("AUTH0_CLIENT_ID"),
		ClientSecret: os.Getenv("AUTH0_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("AUTH0_CALLBACK_URL"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("https://%s/authorize", auth0Domain),
			TokenURL: fmt.Sprintf("https://%s/oauth/token", auth0Domain),
		},
		Scopes: []string{"openid", "profile", "email"},
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	state := "exampleState" // CSRF対策のためのランダムな状態を使用することを推奨
	url := oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state != "exampleState" {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// HttpOnly クッキーをセット
	setAuthCookie(w, token.AccessToken)

	// フロントエンドにリダイレクト
	http.Redirect(w, r, "http://localhost:8000", http.StatusSeeOther)
}

func setAuthCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // ローカル環境ではfalse、プロダクション環境ではtrueに設定
		MaxAge:   3600,  // クッキーの有効期限を1時間に設定
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}

func validateTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")

		if err != nil {

			http.Error(w, "Unauthorized: No token found", http.StatusUnauthorized)
			return
		}

		tokenString := cookie.Value

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {

			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}

			cert, err := getPemCert(token)
			if err != nil {
				return nil, err
			}
			return jwt.ParseRSAPublicKeyFromPEM([]byte(cert))
		})

		if err != nil || !token.Valid {

			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {

	w.Write([]byte("You have accessed a protected resource!"))
}

func getPemCert(token *jwt.Token) (string, error) {
	domain := os.Getenv("AUTH0_DOMAIN")
	certURL := fmt.Sprintf("https://%s/.well-known/jwks.json", domain)

	resp, err := http.Get(certURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var jwks struct {
		Keys []struct {
			Kid string   `json:"kid"`
			X5c []string `json:"x5c"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return "", err
	}

	for _, key := range jwks.Keys {
		if key.Kid == token.Header["kid"] {
			return fmt.Sprintf("-----BEGIN CERTIFICATE-----\n%s\n-----END CERTIFICATE-----", key.X5c[0]), nil
		}
	}
	return "", fmt.Errorf("Unable to find appropriate key")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 必要なCORSヘッダーを設定
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// プリフライトリクエストの場合、早期に終了
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/callback", callbackHandler)
	r.Handle("/protected", validateTokenMiddleware(http.HandlerFunc(protectedHandler)))

	// CORSミドルウェアをラップ
	http.Handle("/", corsMiddleware(r))

	log.Println("Server started at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

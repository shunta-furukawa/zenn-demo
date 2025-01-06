package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
)

// グローバル変数
var oauthConfig *oauth2.Config

// UserInfoResponse は /userinfo エンドポイントのレスポンスを格納する構造体
type UserInfoResponse struct {
	Sub   string `json:"sub"`   // ユーザーID
	Email string `json:"email"` // メールアドレス
	Name  string `json:"name"`  // ユーザー名
}

func init() {
	loadEnvVariables()      // 環境変数をロード
	initializeOAuthConfig() // OAuth2設定を初期化
}

func loadEnvVariables() {
	// .envファイルを読み込む
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

func initializeOAuthConfig() {
	// Auth0のドメインを取得
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	// OAuth2設定を構築
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

func main() {
	r := mux.NewRouter()

	// エンドポイント定義
	r.HandleFunc("/login", loginHandler)                                                // ログイン処理
	r.HandleFunc("/callback", callbackHandler)                                          // コールバック処理
	r.Handle("/protected", validateTokenMiddleware(http.HandlerFunc(protectedHandler))) // 保護されたリソース
	r.HandleFunc("/logout", logoutHandler)                                              // ログアウト処理

	http.Handle("/", corsMiddleware(r)) // CORS設定をミドルウェアで追加

	log.Println("Server started at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// 認証URLを生成しリダイレクト
	state := "exampleState"
	url := oauthConfig.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	// CSRF保護のためのstate確認
	if state := r.URL.Query().Get("state"); state != "exampleState" {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// 認証コードを取得しトークン交換
	code := r.URL.Query().Get("code")
	token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// クッキーにアクセストークンを保存
	setAuthCookie(w, token.AccessToken)
	http.Redirect(w, r, "http://localhost:8000", http.StatusSeeOther)
}

func setAuthCookie(w http.ResponseWriter, token string) {
	// HttpOnlyクッキーを設定
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // ローカル環境ではfalse、本番環境ではtrue
		MaxAge:   3600,  // 有効期限を1時間に設定
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(w, cookie)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	// クッキーを無効化してログアウト処理
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logout successful"))
}

func validateTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// クッキーからトークンを取得
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Error(w, "Unauthorized: No token found", http.StatusUnauthorized)
			return
		}

		// トークンを検証
		userInfo, err := validateOpaqueToken(cookie.Value)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unauthorized: %v", err), http.StatusUnauthorized)
			return
		}

		// 認証成功時のログ
		log.Printf("Authenticated user: %s (%s)", userInfo.Name, userInfo.Email)
		next.ServeHTTP(w, r)
	})
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	// 保護されたリソースにアクセス成功時のレスポンス
	w.Write([]byte("You have accessed a protected resource!"))
}

func validateOpaqueToken(token string) (*UserInfoResponse, error) {
	// Auth0の/userinfoエンドポイントを使用してトークンを検証
	domain := os.Getenv("AUTH0_DOMAIN")
	userinfoURL := fmt.Sprintf("https://%s/userinfo", domain)

	req, err := http.NewRequest("GET", userinfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call /userinfo endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid token: received status %d", resp.StatusCode)
	}

	// ユーザー情報をデコード
	var userInfo UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode /userinfo response: %v", err)
	}

	return &userInfo, nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORSヘッダーを設定
		setCorsHeaders(w)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func setCorsHeaders(w http.ResponseWriter) {
	// 必要なCORSヘッダーを設定
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8000")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

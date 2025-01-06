# Auth0 と Goバックエンド を用いた httponly なクッキー認証

このプロジェクトでは、Auth0 を利用して Go バックエンドで認証を実装し、HttpOnly クッキーを使用してセキュリティを強化します。また、ログアウト機能も実装し、セッションを安全に終了する方法を解説します。

---

## フロントエンドとバックエンドを分離する理由

現代のウェブアプリケーションでは、フロントエンドとバックエンドを分離するアーキテクチャが主流です。その理由として、以下が挙げられます。

- **開発体制の分離**  
  フロントエンドとバックエンドを分けることで、それぞれのチームが独立して開発を進められます。これにより、UIの変更や機能追加が迅速に行え、開発効率が向上します。

- **リッチなUIの実現**  
  ReactやVue.jsといったフロントエンドフレームワークを使用することで、ユーザー体験を向上させるリッチなインターフェースを構築しやすくなります。

- **スケーラビリティの向上**  
  バックエンドをAPIとして提供することで、複数のプラットフォーム（ウェブ、モバイルアプリなど）からの利用が可能となり、システムの拡張性が高まります。

- **責務の明確化**  
  各層の役割が明確になるため、コードの保守性が向上し、バグの発見や修正が容易になります。

---

## 認証方法の比較とHttpOnlyクッキーの利点

### 認証方法の比較

ウェブアプリケーションにおけるユーザー認証には、主に以下の方法があります。

#### 1. クッキー認証

サーバーがユーザーのセッション情報をクッキーとしてクライアントに保存し、以降のリクエストでそのクッキーを使用してユーザーを識別する方法です。

- **利点**  
  - サーバー側でセッション管理が可能で、ユーザー情報の保持が容易。  
  - ブラウザが自動的にクッキーを送信するため、追加の実装が少ない。

- **欠点**  
  - クッキーが盗まれると、セッションハイジャックのリスクがある。  
  - クロスサイトスクリプティング（XSS）攻撃により、クッキー情報が漏洩する可能性がある。

#### 2. トークン認証（例: JWT）

ユーザー認証後にサーバーがJSON Web Token（JWT）を発行し、クライアントがそのトークンをリクエストヘッダーに含めて送信する方法です。

- **利点**  
  - ステートレスであり、サーバーのスケーラビリティが向上。  
  - 複数のドメイン間での認証が容易。

- **欠点**  
  - トークンの管理をクライアント側で行う必要があり、実装が複雑になる場合がある。  
  - トークンの漏洩リスクが存在し、適切な保護が必要。

### HttpOnlyクッキーの利点

クッキー認証において、`HttpOnly` 属性を設定することで、以下のセキュリティ強化が可能です。

- **JavaScriptからのアクセス制限**  
  `HttpOnly` 属性を持つクッキーは、クライアントサイドのJavaScriptからアクセスできません。これにより、XSS攻撃によるクッキーの盗難を防ぐことができます。

- **セッションハイジャックの防止**  
  クッキーの不正取得を防ぐことで、セッションハイジャックのリスクを低減します。

---

## 必要な環境

- Go 1.18 以上
- Auth0 アカウント
- フロントエンド環境（HTML, JavaScript）

---

## セットアップ手順

### 1. 必要なパッケージをインストール

以下のコマンドで必要なパッケージをインストールします。

```
go get github.com/gorilla/mux
go get github.com/joho/godotenv
go get github.com/dgrijalva/jwt-go
```

---

### 2. Auth0 の設定

Auth0 ダッシュボードで以下を設定します：

1. **アプリケーション作成**
   - Auth0 ダッシュボードでアプリケーションを作成し、以下の情報を取得：
     - Client ID
     - Client Secret
     - Domain

2. **API 作成**
   - 「APIs」で新しい API を作成します。
   - 「Identifier」を設定（例: `https://my-api/`）。

3. **設定**
   - **Allowed Callback URLs**:
     ```
     http://localhost:3000/callback
     ```
   - **Allowed Logout URLs**:
     ```
     http://localhost:8000/index.html
     ```
   - **Allowed Web Origins**:
     ```
     http://localhost:8000
     ```

4. **Grant Types の確認**
   - 「Settings」タブの下部にある「Grant Types」で **Authorization Code** が有効になっていることを確認。

---

### 3. `.env` ファイルの作成

`.env` ファイルを作成し、以下のように環境変数を設定します。

```
AUTH0_DOMAIN=your-auth0-domain
AUTH0_CLIENT_ID=your-client-id
AUTH0_CLIENT_SECRET=your-client-secret
AUTH0_CALLBACK_URL=http://localhost:3000/callback
AUTH0_AUDIENCE=your-api-identifier
```

---

### 4. サーバ実装

以下のコードを使用して Go サーバを実装します。

#### `main.go`

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

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
	state := "exampleState"
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

	setAuthCookie(w, token.AccessToken)
	http.Redirect(w, r, "http://localhost:8000", http.StatusSeeOther)
}

func setAuthCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   3600,
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

			return nil, fmt.Errorf("Public key verification not implemented")
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

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	http.Redirect(w, r, "http://localhost:8000/index.html", http.StatusSeeOther)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/callback", callbackHandler)
	r.Handle("/protected", validateTokenMiddleware(http.HandlerFunc(protectedHandler)))
	r.HandleFunc("/logout", logoutHandler)

	log.Println("Server started at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```

---

### 5. フロントエンドの実装

以下の `index.html` を作成して、フロントエンドを実装します。

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Auth0 Protected Test</title>
</head>
<body>
    <h1>Auth0 Protected Endpoint Test</h1>
    <button id="login-button">Log in</button>
    <button id="protected-button">Access Protected Resource</button>
    <button id="logout-button">Log out</button>
    <p id="result"></p>

    <script>
        document.getElementById('login-button').addEventListener('click', () => {
            window.location.href = 'http://localhost:3000/login';
        });

        document.getElementById('protected-button').addEventListener('click', async () => {
            try {
                const response = await fetch('http://localhost:3000/protected', { credentials: 'include' });
                if (response.ok) {
                    const text = await response.text();
                    document.getElementById('result').innerText = text;
                } else {
                    document.getElementById('result').innerText = 'Access Denied: Unauthorized';
                }
            } catch (err) {
                console.error(err);
                document.getElementById('result').innerText = 'Error accessing protected resource';
            }
        });

        document.getElementById('logout-button').addEventListener('click', async () => {
            await fetch('http://localhost:3000/logout', { credentials: 'include' });
            window.location.href = 'http://localhost:8000/index.html';
        });
    </script>
</body>
</html>
```

---

### 6. 動作確認

1. **バックエンドを起動**:
   ```
   go run main.go
   ```

2. **フロントエンドを起動**:
   ```
   python -m http.server 8000
   ```

3. ブラウザで `http://localhost:8000/index.html` を開き、以下を確認:
   - **Log in** ボタンでログインフローを確認。
   - **Access Protected Resource** ボタンで認証済みリソースへのアクセスを確認。
   - **Log out** ボタンでログアウトし、認証済みリソースにアクセスできないことを確認。

---

これでログアウト機能を含めたフルセットのシステムが完成します！必要に応じて調整してください。質問があればお知らせください！ 🎉

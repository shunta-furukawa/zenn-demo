
# Auth0を活用したGoバックエンドでの認証とHttpOnlyクッキー管理

この記事では、Auth0を利用してGoバックエンドでの認証を実装し、HttpOnlyクッキーを使用してセキュリティを強化する方法を解説します。また、認証前後でバックエンドの保護されたエンドポイントにアクセスできるようになる流れを、フロントエンドと併せて説明します。

---

## 実装手順

### 必要なパッケージの準備

以下のコマンドで必要なパッケージをインストールします。

```
go get github.com/gorilla/mux
go get github.com/joho/godotenv
go get github.com/dgrijalva/jwt-go
```

---

### Auth0の設定

Auth0のダッシュボードで以下の設定を行います：

- **APIの作成**: API識別子（audience）を設定
- **アプリケーションの作成**: クライアントIDとクライアントシークレットを取得

次に、`.env`ファイルを作成し、以下のように環境変数を設定します。

```
AUTH0_DOMAIN=your-auth0-domain
AUTH0_CLIENT_ID=your-client-id
AUTH0_CLIENT_SECRET=your-client-secret
AUTH0_CALLBACK_URL=http://localhost:3000/callback
AUTH0_AUDIENCE=your-api-identifier
```

---

### Goサーバの実装

以下は、Auth0を利用したGoサーバのコードで、認証後に`protected`エンドポイントがアクセス可能になる流れを含んでいます。

```
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(token)
}

func setAuthCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // ローカル開発ではfalse。プロダクションではtrueに設定
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

			return nil, fmt.Errorf("Public key verification not implemented") // 実際には公開鍵取得の処理を実装
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

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/callback", callbackHandler)
	r.Handle("/protected", validateTokenMiddleware(http.HandlerFunc(protectedHandler)))
	http.Handle("/", r)

	log.Println("Server started at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
```

---

### フロントエンドの実装

以下は、簡単なフロントエンドです。認証前後のアクセスを確認するために利用します。このコードを`index.html`として保存します。

```
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
    </script>
</body>
</html>
```

---

### フロントエンド用ローカルサーバの立ち上げ

1. **Pythonを使用してローカルサーバを起動する場合**  
   以下のコマンドを実行して、ローカルサーバを起動します：

   ```
   python -m http.server 8000
   ```

   これにより、現在のディレクトリが`http://localhost:8000`で配信されます。

2. **Node.jsを使用してサーバを起動する場合**  
   `http-server`モジュールを使う場合：

   ```
   npm install -g http-server
   http-server -p 8000
   ```

   `http://localhost:8000` にアクセスします。

---

### ローカルでの動作確認

1. **Goサーバを起動する**  
   以下のコマンドを使用してGoサーバを起動します：

   ```
   go run main.go
   ```

2. **フロントエンドサーバを起動する**  
   PythonやNode.jsの方法で`http://localhost:8000`を起動します。

3. **ログインフローの確認**  
   - ブラウザで`http://localhost:8000/index.html`を開きます。
   - 「Log in」ボタンをクリックしてAuth0のログインページにリダイレクトされます。
   - ログイン後、Goサーバがトークンを発行し、クッキーに保存します。

4. **保護されたエンドポイントへのアクセス**  
   - 「Access Protected Resource」をクリックします。
   - 認証前は「Access Denied: Unauthorized」と表示されます。
   - 認証後は「You have accessed a protected resource!」と表示されます。

---

## 結論

この記事では、Auth0を利用してGoバックエンドの認証フローを構築し、フロントエンドと連携する完全なシステムを作成しました。ローカルサーバを使用してセキュリ

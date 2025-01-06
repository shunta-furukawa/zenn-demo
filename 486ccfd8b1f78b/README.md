# GoとAuth0で実現するセキュアな認証: HttpOnlyクッキーの活用法

現代のウェブアプリケーションでは、セキュリティを確保しながらスムーズなユーザー体験を提供することが求められています。特に、ユーザー認証はアプリケーションの中核を担う部分であり、セキュリティリスクを最小限に抑えることが重要です。

本記事では、HttpOnlyクッキーを利用した認証の実装方法を中心に解説します。この方法は、クロスサイトスクリプティング（XSS）攻撃を防ぎ、トークンを安全に管理する優れたアプローチです。さらに、Auth0を活用して認証フローを簡略化し、Go言語を用いたバックエンド実装を通じて、セキュリティと使いやすさを両立したシステムを構築する方法を紹介します。

この記事を読むことで、以下のことが学べます：

- フロントエンドとバックエンドを分離する理由とその利点
- HttpOnlyクッキーを使用することで実現できるセキュリティ向上のポイント
- Auth0を利用したGoバックエンドでの認証フローの実装方法
- ログイン・ログアウトフローの構築と安全なセッション管理

認証に関する基礎知識から実装方法まで、ステップバイステップで説明していきます。特に「セキュアでシンプルな認証」を目指す方にとって、すぐに使える具体的なサンプルコードと設定例を提供します。それでは、一緒に進めていきましょう！

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

ここまでで、フロントエンドとバックエンドを分離する理由や、さまざまな認証方法の特徴とHttpOnlyクッキーを活用する利点について解説しました。それでは次に、これらの理論を踏まえた実際の実装方法について解説していきます。

このプロジェクトでは、以下のポイントを押さえながら、Go言語を使用してAuth0を活用した認証システムを構築します。

- **Auth0を活用したシンプルでセキュアな認証の実現** 
Auth0は、認証やユーザー管理を容易にするプラットフォームであり、アクセストークンやIDトークンの発行を簡単に行えます。これを活用し、安全で拡張性の高い認証システムを構築します。

- **HttpOnlyクッキーによるトークン管理** 
認証の際に発行されたアクセストークンをHttpOnlyクッキーに保存し、セキュリティを強化します。クッキーはブラウザが自動的に送信するため、APIリクエストの実装もシンプルになります。

- **セッションの安全な終了（ログアウト機能）** 
セッションを終了する際にクッキーを削除することで、ログアウト処理を安全かつ確実に行います。

これから、具体的なコードを用いて、Goを使ったバックエンドの実装方法をステップバイステップで説明していきます。また、Auth0の設定方法についても併せて解説しますので、一緒に進めていきましょう。

ではまず、プロジェクトのセットアップと、必要なパッケージのインストールから始めます！

---

## 必要な環境

- Go 1.18 以上
- Auth0 アカウント
- フロントエンド環境（HTML, JavaScript）

---

## セットアップ手順

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
            try {
                const response = await fetch('http://localhost:3000/logout', {
                    method: 'POST', // POST リクエストを送信
                    credentials: 'include', // クッキーを送信
                });

                if (response.ok) {
                    // ログアウト成功後にリダイレクト
                    document.getElementById('result').innerText = 'Logout successful!';
                    setTimeout(() => {
                        window.location.href = 'http://localhost:8000/index.html';
                    }, 1000); // 1秒後にリダイレクト
                } else {
                    document.getElementById('result').innerText = 'Logout failed!';
                }
            } catch (err) {
                console.error('Error during logout:', err);
                document.getElementById('result').innerText = 'Error during logout.';
            }
        });
    </script>
</body>

</html>
```

---

### 6. 動作確認

まず、以下の準備を完了してください：

1. **バックエンドの起動**:
```
go run main.go
```

2. **フロントエンドの起動**:
```
python -m http.server 8000
```

3. ブラウザで `http://localhost:8000/index.html` を開きます。

---

#### **6.1. 初期状態**

ブラウザでフロントエンドにアクセスすると、以下のような画面が表示されます。

- **初期状態**: ログインボタンのみが機能します。保護されたリソースやログアウトボタンを押してもアクセスできません。

![](https://storage.googleapis.com/zenn-user-upload/4b067d84989a-20250106.png)

---

#### **6.2. ログイン画面**

ログインボタンをクリックすると、Auth0 のログイン画面にリダイレクトされます。

- **ログイン画面**: ユーザー名とパスワードを入力して認証を進めます。

![](https://storage.googleapis.com/zenn-user-upload/ac88c0047bfe-20250106.png)

---

#### **6.3. ログイン成功後の画面**

ログインが成功すると、ブラウザに戻り、以下のように保護されたリソースにアクセス可能になります。

- **ログイン成功後**:  
  - 保護されたリソース（"You have accessed a protected resource!"）が正常に表示されます。
  - `Access Protected Resource` ボタンが機能することを確認できます。

![](https://storage.googleapis.com/zenn-user-upload/c9f5ad5e1810-20250106.png)

---

#### **6.4. ログアウト処理**

`Log out` ボタンをクリックすると、以下のようにログアウトが成功し、再び保護されたリソースにアクセスできなくなります。

- **ログアウト成功後**:  
  - ログアウトメッセージ（"Logout successful"）が表示されます。
  - 再度リソースにアクセスしようとすると、エラーが表示されます。

![](https://storage.googleapis.com/zenn-user-upload/73d59f940c5d-20250106.png)

---

#### **6.5. 未ログイン状態で保護されたリソースにアクセスした場合**

ログアウト後、`Access Protected Resource` ボタンをクリックすると、以下のエラーが表示されます。

- **エラー画面**:  
  - "Access Denied: Unauthorized" というメッセージが画面に表示されます。

![](https://storage.googleapis.com/zenn-user-upload/9b394ab94dc6-20250106.png)

---

### 7. HttpOnly クッキーをブラウザで確認する方法

HttpOnly クッキーは通常の JavaScript からアクセスできませんが、ブラウザの開発者ツールを使用して確認できます。（以下、ログインした状態で行ってください）

#### **手順: Chrome の場合**

1. **開発者ツールを開く**:
   - キーボードショートカットで開く: `Ctrl + Shift + I` (Windows) または `Cmd + Option + I` (Mac)。
   - または、ブラウザの右上メニューから「その他のツール > デベロッパーツール」を選択。

2. **「Application」タブを選択**:
   - 開発者ツールの上部メニューから「Application」をクリックします。

3. **「Cookies」セクションを選択**:
   - 左側のサイドメニューから「Storage > Cookies」をクリックし、`http://localhost:8000` を選択します。

4. **クッキー情報を確認**:
   - `auth_token` という名前のクッキーが保存されていることを確認できます。
   - `HttpOnly` 属性が有効であることを確認してください（通常、この属性は列として表示されます）。
   
![](https://storage.googleapis.com/zenn-user-upload/bd7e9c32eb68-20250106.png)

---

#### **HttpOnly クッキーが JS からアクセスできないことを確認する**

以下のスニペットをブラウザのコンソールで実行してみてください：

```
console.log(document.cookie);
```

- 結果: `auth_token` が含まれていないことを確認できます。
  - これは `HttpOnly` 属性により、JavaScript からクッキーにアクセスできないためです。

![](https://storage.googleapis.com/zenn-user-upload/1e6905c133a7-20250106.png)

---

これらの手順を通じて、Auth0 と Go バックエンドを用いた HttpOnly クッキー認証の動作を確認できます。

---


ここまで、Auth0を活用したGoバックエンドでの認証フローと、HttpOnlyクッキーを利用したセキュアなトークン管理について詳しく解説しました。

ウェブアプリケーションにおける認証は、ユーザー体験の向上だけでなく、セキュリティを確保する上でも重要な要素です。本記事で紹介したHttpOnlyクッキーを利用した方法は、JavaScriptによる不正アクセスを防ぎつつ、シンプルで堅牢な認証システムを構築するための一例です。

また、Auth0を活用することで、認証機能を自前で実装する手間を大幅に削減しつつ、拡張性の高い認証基盤を構築できます。加えて、ログインやログアウトフローのセキュアな実装や、トークン管理の仕組みを理解することで、より安全で使いやすいシステム設計を目指せるでしょう。

最後に、本記事の内容を通じて、読者の皆さまがセキュアで実用的な認証システムを構築するヒントを得られたのであれば幸いです。ぜひ今回のサンプルコードを参考に、実際のプロジェクトで試してみてください！

もし本記事に関してご質問やフィードバックがあれば、ぜひコメント欄でお知らせください。また、この記事が役立った場合はシェアしていただけると励みになります。それでは、セキュアなアプリケーション開発を楽しんでください！ 😊

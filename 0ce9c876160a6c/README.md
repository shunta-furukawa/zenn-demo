# 【OpenAI Dev Day 2024】Realtime API × WebRTC でブラウザボイスチャットを作ってみた

OpenAI Dev Day 2024 で発表された「Realtime API」を使って、**ブラウザだけ**でボイスチャットを実現するデモを作ってみました。本記事では、WebRTC を活用した理由や JavaScript 実装のポイントを解説します。

## 背景
OpenAI の Realtime API は、GPT-4 系のモデルと **リアルタイムな音声・テキストのやり取り** が可能になる新しい API です。  
これまではテキストベースの対話が中心でしたが、音声の送受信をブラウザで直接扱えるようになると、**スマートスピーカー的な体験**や **音声アシスタント** などを簡単に実装できるようになります。

### WebRTC を選んだ理由
実は Realtime API への接続には **WebSocket** 経由の方法もあります。  
- WebSocket はシンプルに通信できますが、**音声ストリームを送受信するときにサーバー実装を挟む**必要が出てきます。(サーバーが音声の転送を仲介するケース)  
- 一方、**WebRTC** はブラウザ同士(またはブラウザとサーバー)の **点対点(P2P)接続**が可能で、さらに **音声ストリーム** に最適化された仕組みが標準で備わっています。

本記事のデモでは、**できる限りサーバーレス**で手軽に試したかったこともあり、WebRTC を使う構成を採用しています。

## Realtime API のざっくりした仕組み
- **WebRTC の SDP** (Session Description Protocol) を用いて、ブラウザの `RTCPeerConnection` と OpenAI 側の Realtime API が相互にストリーム情報を交換します。  
- 接続が確立すると、音声やテキストのイベントを**リアルタイム**にやりとりできるようになります。  
- **音声ストリーム**: `navigator.mediaDevices.getUserMedia` で取得したマイク音声を送信し、サーバー側（モデル）が処理した結果を受信します。  
- **テキスト・イベント**: `DataChannel` を通じて JSON メッセージを送受信します。  
  - 送信例:  
    ```
    {
      "type": "response.create",
      "response": {
        "modalities": ["text"],
        "instructions": "こんにちは、自己紹介して。"
      }
    }
    ```
  - 受信例:  
    ```
    {
      "type": "response",
      "output": {
        "text": "こんにちは！私は GPT-4o ...",
        ...
      }
    }
    ```

これらのメッセージは、**「どういう出力を生成してほしいか (text, audio, function call... )」** といった指示を**リクエスト**し、サーバー側のモデルがそれを**レスポンス**として返すイメージです。  
「テキストは一行ずつ逐次受け取りたい」「音声も途中から再生したい」など、細かいコントロールができるのが Realtime API の強みです。

---

## サンプル: 単一 HTML + JS で試せるデモ

以下のサンプルは、**ブラウザで「通常の API キー」を直接入力**し、エフェメラルキーを取得してから WebRTC 接続するというものです。  
本番でこのやり方をするのは危険ですが、**学習用・ローカル検証**としては気軽に動かせるので便利です。

```html
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="UTF-8">
  <title>OpenAI Realtime API + WebRTC デモ (トークスクリプト付き)</title>
  <style>
    body { font-family: sans-serif; margin: 20px; }
    #log, #transcript {
      white-space: pre-wrap;
      background: #f4f4f4;
      padding: 1em;
      border-radius: 5px;
      margin-top: 1em;
      height: 10em;
      overflow-y: auto;
    }
    button {
      margin: 0.3em 0;
      padding: 0.3em 0.7em;
    }
    .error { color: red; }
    .info { color: blue; }
    .userMsg { font-weight: bold; }
    .assistantMsg { color: green; }
  </style>
</head>
<body>
  <h1>OpenAI Realtime API + WebRTC デモ (サーバーレス＋トランスクリプト)</h1>
  <p>
    このデモはローカル学習用です。<br>
    1. APIキー入力 → <strong>エフェメラルキー取得</strong><br>
    2. <strong>接続開始</strong> → マイク利用を許可<br>
    3. テキストを入力して「送信」→ Realtime API が応答 (DataChannel 経由)<br>
    4. 必要に応じて「接続終了」で WebRTC をクローズ
  </p>

  <!-- 1. APIキー入力欄 -->
  <label for="inputApiKey"><strong>OpenAI API Key:</strong></label><br>
  <input type="password" id="inputApiKey" size="60" placeholder="sk-xxxx..." />
  <br><br>

  <!-- 2. 操作用ボタン -->
  <button id="btnGetEphemeral">1. エフェメラルキー取得</button>
  <button id="btnStartConnection" disabled>2. 接続開始</button>
  <button id="btnEndConnection" disabled>接続終了</button>
  <br>

  <!-- 3. エフェメラルキー表示 -->
  <p><strong>エフェメラルキー:</strong> <span id="ephemeralToken">（未取得）</span></p>

  <!-- リモート音声再生用 -->
  <audio id="remoteAudio" autoplay></audio>

  <!-- ログ表示領域 -->
  <div id="log"></div>

  <!-- 入出力トランスクリプト領域 -->
  <h3>入力・出力トランスクリプト</h3>
  <div id="transcript"></div>

  <!-- ユーザー入力フォーム -->
  <p>
    <input type="text" id="userInput" size="50" placeholder="メッセージを入力"/>
    <button id="btnSend" disabled>送信</button>
  </p>

  <script>
    // UI要素を取得
    const inputApiKey = document.getElementById("inputApiKey");
    const btnGetEphemeral = document.getElementById("btnGetEphemeral");
    const btnStartConnection = document.getElementById("btnStartConnection");
    const btnEndConnection = document.getElementById("btnEndConnection");
    const ephemeralTokenEl = document.getElementById("ephemeralToken");
    const remoteAudio = document.getElementById("remoteAudio");
    const logEl = document.getElementById("log");
    const transcriptEl = document.getElementById("transcript");
    const userInputEl = document.getElementById("userInput");
    const btnSend = document.getElementById("btnSend");

    // 内部状態
    let ephemeralKey = null;           // 取得したエフェメラルキー
    let pc = null;                     // RTCPeerConnection
    let dc = null;                     // DataChannel
    let localStream = null;            // マイク音声 (MediaStream)
    let connectionActive = false;      // 接続中かどうか

    // ▼▼ ログ表示用ヘルパー ▼▼
    function log(...msgs) {
      console.log(...msgs);
      logEl.textContent += msgs.join(" ") + "\n";
    }
    function logError(...msgs) {
      console.error(...msgs);
      logEl.innerHTML += `<span class="error">${msgs.join(" ")}</span>\n`;
    }
    function logInfo(...msgs) {
      console.info(...msgs);
      logEl.innerHTML += `<span class="info">${msgs.join(" ")}</span>\n`;
    }

    // ▼▼ トランスクリプトに表示する ▼▼
    function addTranscript(message, sender = "system") {
      // sender: "user" | "assistant" | "system" など
      let className = "";
      if (sender === "user") className = "userMsg";
      if (sender === "assistant") className = "assistantMsg";
      transcriptEl.innerHTML += `<div class="${className}">[${sender}] ${message}</div>`;
      transcriptEl.scrollTop = transcriptEl.scrollHeight; // スクロール下まで移動
    }

    // 1. エフェメラルキー取得ボタン
    btnGetEphemeral.addEventListener("click", async () => {
      const apiKey = inputApiKey.value.trim();
      if (!apiKey) {
        logError("APIキーを入力してください。");
        return;
      }
      log("エフェメラルキーを取得します...");

      try {
        // Realtime Sessions エンドポイントに直接リクエストしてエフェメラルキー取得
        const res = await fetch("https://api.openai.com/v1/realtime/sessions", {
          method: "POST",
          headers: {
            "Authorization": `Bearer ${apiKey}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            model: "gpt-4o-realtime-preview-2024-12-17",
            // voice: "verse", // 音声合成の種類など必要に応じて指定
          }),
        });

        if (!res.ok) {
          const errText = await res.text();
          logError("エフェメラルキー取得失敗:", res.status, res.statusText, errText);
          ephemeralKey = null;
          ephemeralTokenEl.textContent = "取得失敗";
          btnStartConnection.disabled = true;
          return;
        }

        const data = await res.json();
        if (!data.client_secret || !data.client_secret.value) {
          logError("レスポンスに client_secret.value がありません:", JSON.stringify(data));
          ephemeralKey = null;
          ephemeralTokenEl.textContent = "取得失敗";
          btnStartConnection.disabled = true;
          return;
        }

        ephemeralKey = data.client_secret.value;
        ephemeralTokenEl.textContent = ephemeralKey;
        log("エフェメラルキー取得成功:", ephemeralKey);
        btnStartConnection.disabled = false;
      } catch (err) {
        logError("エフェメラルキー取得中にエラー:", err);
      }
    });

    // 2. 接続開始ボタン
    btnStartConnection.addEventListener("click", async () => {
      if (!ephemeralKey) {
        logError("エフェメラルキーがありません。先に「エフェメラルキー取得」を行ってください。");
        return;
      }
      log("WebRTC 接続開始...");

      // RTCPeerConnection 作成
      pc = new RTCPeerConnection();

      // リモートから音声トラックが届いたら再生
      pc.ontrack = (event) => {
        log("ontrack: リモート音声ストリームを受信");
        remoteAudio.srcObject = event.streams[0];
      };

      // 接続状態変化の監視 (任意)
      pc.onconnectionstatechange = () => {
        log("PeerConnection state:", pc.connectionState);
      };

      // マイク取得
      try {
        localStream = await navigator.mediaDevices.getUserMedia({ audio: true });
        localStream.getTracks().forEach(track => pc.addTrack(track, localStream));
        log("マイク入力を取得しました。");
      } catch (err) {
        logError("マイクへのアクセスが拒否されました:", err);
        return;
      }

      // DataChannel 作成
      dc = pc.createDataChannel("oai-events");
      dc.addEventListener("open", () => log("DataChannel open"));
      dc.addEventListener("close", () => log("DataChannel close"));
      dc.addEventListener("message", (e) => {
        try {
          // 受信したメッセージが JSON ならパースする
          const data = JSON.parse(e.data);
          log("DataChannel 受信 (JSON):", data);

          // ここでは "response" 系のイベントが来たら「アシスタントメッセージ」として表示してみる
          if (data.type && data.type.startsWith("response")) {
            // 例: text をまとめて受け取るパターン (サマリーメッセージ等)
            //     もしくは、部分的なトランスクリプトなど…
            if (data.output && data.output.text) {
              addTranscript(data.output.text, "assistant");
            } else {
              // そのほか適宜処理
              addTranscript(JSON.stringify(data), "assistant");
            }
          } else {
            // 通常ログ
            addTranscript(JSON.stringify(data), "assistant");
          }
        } catch (_) {
          // JSON parse エラーなどの場合はそのまま表示
          log("DataChannel 受信 (text):", e.data);
          addTranscript(e.data, "assistant");
        }
      });

      // SDP オファーを作成
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      log("SDP オファー作成:", offer);

      // Realtime API へ送信して SDP アンサーを取得
      try {
        const baseUrl = "https://api.openai.com/v1/realtime";
        const model = "gpt-4o-realtime-preview-2024-12-17";
        const sdpResponse = await fetch(`${baseUrl}?model=${model}`, {
          method: "POST",
          body: offer.sdp,
          headers: {
            "Authorization": `Bearer ${ephemeralKey}`,
            "Content-Type": "application/sdp",
          },
        });
        if (!sdpResponse.ok) {
          const errText = await sdpResponse.text();
          logError("SDP送信エラー:", sdpResponse.status, sdpResponse.statusText, errText);
          return;
        }
        const answerSDP = await sdpResponse.text();
        await pc.setRemoteDescription({ type: "answer", sdp: answerSDP });
        logInfo("SDP アンサーを適用しました。WebRTC 接続完了。");
        connectionActive = true;

        // ボタンの状態を更新
        btnEndConnection.disabled = false;
        btnSend.disabled = false;
      } catch (err) {
        logError("SDP通信エラー:", err);
      }
    });

    // 「接続終了」ボタン
    btnEndConnection.addEventListener("click", () => {
      endConnection();
    });

    function endConnection() {
      if (dc) {
        dc.close();
        dc = null;
      }
      if (pc) {
        pc.close();
        pc = null;
      }
      if (localStream) {
        localStream.getTracks().forEach(track => track.stop());
        localStream = null;
      }
      connectionActive = false;
      btnEndConnection.disabled = true;
      btnSend.disabled = true;
      logInfo("WebRTC 接続を終了しました。");
    }

    // 送信ボタン (テキスト送信)
    btnSend.addEventListener("click", () => {
      if (!connectionActive || !dc || dc.readyState !== "open") {
        logError("DataChannel が開いていません。接続を開始してください。");
        return;
      }
      const text = userInputEl.value.trim();
      if (!text) return;

      // ユーザーメッセージとしてトランスクリプトに追加
      addTranscript(text, "user");
      userInputEl.value = "";

      // Realtime API へのイベント送信例 (response.create)
      // ここではテキストモードで「instructions」に入れていますが、
      // 実際には API 仕様に応じて調整してください。
      const eventMsg = {
        type: "response.create",
        response: {
          modalities: ["text"],
          instructions: text
        },
      };

      dc.send(JSON.stringify(eventMsg));
      log("DataChannel 送信:", eventMsg);
    });
  </script>
</body>
</html>

```

---

### 実行方法

1. 上記の ```HTML``` をファイルとして保存し、**HTTPS または localhost** で提供します。  
2. ブラウザで開いて ```OpenAI API Key``` 欄に自分の通常キー (sk-xxxx...) を入力。  
3. **「1. エフェメラルキー取得」** をクリック → 取得成功で短期的に有効なキーが表示されます。  
4. **「2. 接続開始」** → マイクアクセスを許可してください。  
5. 下部でテキストを送信 → データチャンネル経由でモデルが応答、音声も自動で再生されます。  
6. 「接続終了」でクローズ。

---

## まとめと注意点

- **WebRTC** を使うことで、ブラウザのみで音声ストリームがやり取りできるのが大きな利点です。  
- 同じ Realtime API を **WebSocket** で扱う場合、サーバーを立てる必要があったりしますが、その場合は WebRTC の手順を省略できる代わりにストリーム伝送の実装が少し複雑化するかもしれません。  
- このデモのように **「通常の API キーをブラウザに直接入力する」** のは、**本番ではセキュリティ的に非常に危険**です。必ずサーバー側でキーを保持し、**エフェメラルキーだけ**をクライアントに渡す構成をとってください。

OpenAI Realtime API は **リアルタイム音声認識・応答**、**関数呼び出し**、**逐次テキスト生成**など、非常に面白い可能性を秘めています。  
ぜひ、今回の簡易デモをベースに、自分だけの **高度なボイスチャット** や **音声アプリ** を作ってみてください！

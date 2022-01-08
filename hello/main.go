package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"unicode/utf8"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/linebot"
)

func main() {
	// ハンドラの登録
	http.HandleFunc("/callback", lineHandler)

	// HTTPサーバーを起動
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

func lineHandler(w http.ResponseWriter, r *http.Request) {
	err := godotenv.Load(os.Getenv("GO_ENV"))
	if err != nil {
		// .env読めなかった場合の処理
		log.Fatal(err)
	}
	secret := os.Getenv("LINE_BOT_CHANNEL_SECRET")
	token := os.Getenv("LINE_BOT_CHANNEL_TOKEN")

	// BOTを初期化
	bot, err := linebot.New(
		secret,
		token,
	)
	if err != nil {
		log.Fatal(err)
	}

	// リクエストからBOTのイベントを取得
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}
	for _, event := range events {
		// イベントがメッセージの受信だった場合
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			// メッセージがテキスト形式の場合
			case *linebot.TextMessage:
				replyMessage := message.Text
				_, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(replyMessage)).Do()
				if err != nil {
					log.Print(err)
				}
			case *linebot.LocationMessage:
				sendRestoInfo(bot, event)
			}
			// 他にもスタンプや画像、位置情報などいろいろ受信できる
		}
	}
}

func sendRestoInfo(bot *linebot.Client, e *linebot.Event) {
	msg := e.Message.(*linebot.LocationMessage)

	lat := strconv.FormatFloat(msg.Latitude, 'f', 2, 64)
	lng := strconv.FormatFloat(msg.Longitude, 'f', 2, 64)

	replyMsg := getRestoInfo(lat, lng)

	res := linebot.NewTemplateMessage(
		"レストラン一覧",
		linebot.NewCarouselTemplate(replyMsg...).WithImageOptions("rectangle", "cover"),
	)

	log.Println(res)

	_, err := bot.ReplyMessage(e.ReplyToken, res).Do()
	if err != nil {
		log.Print("hit")
		log.Print(err)
	}
}

// response APIレスポンス
type response struct {
	Results results `json:"results"`
}

// results APIレスポンスの内容
type results struct {
	Shop []shop `json:"shop"`
}

// shop レストラン一覧
type shop struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Photo   photo  `json:"photo"`
	URLS    urls   `json:"urls"`
}

// photo 写真URL一覧
type photo struct {
	Mobile mobile `json:"mobile"`
}

// mobile モバイル用の写真URL
type mobile struct {
	L string `json:"l"`
}

// urls　URL一覧
type urls struct {
	PC string `json:"pc"`
}

func getRestoInfo(lat string, lng string) []*linebot.CarouselColumn {
	apikey := os.Getenv("API_KEY")

	url := fmt.Sprintf("https://webservice.recruit.co.jp/hotpepper/gourmet/v1/?format=json&key=%s&lat=%s&lng=%s", apikey, lat, lng)

	// リクエストしてボディを取得
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	var data response
	if err := json.Unmarshal(body, &data); err != nil {
		log.Fatal(err)
	}

	var ccs []*linebot.CarouselColumn
	for _, shop := range data.Results.Shop {
		addr := shop.Address
		if 60 < utf8.RuneCountInString(addr) {
			addr = string([]rune(addr)[:60])
		}

		cc := linebot.NewCarouselColumn(
			shop.Photo.Mobile.L,
			shop.Name,
			addr,
			linebot.NewURIAction("ホットペッパーで開く", shop.URLS.PC),
		).WithImageOptions("#FFFFFF")
		ccs = append(ccs, cc)
	}
	return ccs
}

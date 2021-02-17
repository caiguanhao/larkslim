package lark_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/caiguanhao/lark-slim"
)

func TestAPI(t *testing.T) {
	appId := os.Getenv("LARK_APP_ID")
	appSecret := os.Getenv("LARK_APP_SECRET")
	l := lark.API{
		AppId:     appId,
		AppSecret: appSecret,
	}
	_, err := l.GetAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("GetAccessToken() passed")
	chats, err := l.ListAllChats()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ListAllChats() passed")
	if len(chats) > 0 {
		chat, err := l.GetChatInfo(chats[0].ChatId)
		if err != nil {
			t.Fatal(err)
		}
		t.Log("GetChatInfo() passed")
		if len(chat.Members) > 0 {
			user := chat.Members[0].OpenId
			users := []string{user}
			_, err := l.GetUserInfo(users)
			if err != nil {
				t.Fatal(err)
			}
			err = l.AddUsersToChat(chat.ChatId, users)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("AddUsersToChat() passed")
			key, err := l.UploadMessageImage(randomImage())
			if err != nil {
				t.Fatal(err)
			}
			t.Log("UploadMessageImage() passed")
			err = l.SendImageMessage(user, key)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("SendImageMessage() passed")
		}
	}
}

func ExamplePost() {
	post := lark.Post{
		"zh_cn": lark.PostOfLocale{
			Title: "post",
			Content: lark.PostLines{
				{
					{
						Tag:  "text",
						Text: "Name: ",
					},
					{
						Tag:  "a",
						Text: "Hello",
						Href: "https://www.google.com",
					},
				},
			},
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	enc.Encode(post)
	// Output:
	// {
	// 	"zh_cn": {
	// 		"title": "post",
	// 		"content": [
	// 			[
	// 				{
	// 					"tag": "text",
	// 					"text": "Name: "
	// 				},
	// 				{
	// 					"tag": "a",
	// 					"text": "Hello",
	// 					"href": "https://www.google.com"
	// 				}
	// 			]
	// 		]
	// 	}
	// }
}

func randomImage() io.Reader {
	rand.Seed(time.Now().Unix())
	n := rand.Perm(200)
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for i := 0; i < len(n)/2; i++ {
		img.Set(n[2*i], n[2*i+1], color.RGBA{255, 0, 0, 255})
	}
	var w bytes.Buffer
	err := png.Encode(&w, img)
	if err != nil {
		panic(err)
	}
	return &w
}

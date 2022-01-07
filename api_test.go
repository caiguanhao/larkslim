package larkslim_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/caiguanhao/larkslim"
)

func TestAPI(t *testing.T) {
	appId := os.Getenv("LARK_APP_ID")
	appSecret := os.Getenv("LARK_APP_SECRET")
	l := larkslim.API{
		AppId:     appId,
		AppSecret: appSecret,
	}

	chats, err := l.ListAllChats()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("ListAllChats() passed, chats:", len(chats))
	if len(chats) == 0 {
		t.Log("no chats to test")
		return
	}
	t.Log("using chat:", chats[0].Name)
	chat, err := l.GetChatInfo(chats[0].ChatId)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("GetChatInfo() passed")
	if len(chat.Members) == 0 {
		t.Log("no chat members to test")
		return
	}

	var wg sync.WaitGroup
	for i := 0; i < len(chat.Members); i++ {
		wg.Add(1)
		openId := chat.Members[i].OpenId
		go func() {
			defer wg.Done()
			userInfo, err := l.GetUserInfo(openId)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("user info:", userInfo)
		}()
	}
	wg.Wait()

	user := os.Getenv("LARK_OPENID")
	if user == "" {
		t.Log("Set LARK_OPENID env to run more tests")
		return
	}

	userInfo, err := l.GetUserInfo(user)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("user info:", userInfo)

	users := []string{user}
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

func ExamplePost() {
	post := larkslim.Post{
		"zh_cn": larkslim.PostOfLocale{
			Title: "post",
			Content: larkslim.PostLines{
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

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/caiguanhao/larkslim"
)

func die(a ...interface{}) {
	fmt.Fprintln(os.Stderr, a...)
	os.Exit(1)
}

func main() {
	var appId, appSecret, imageType, sendTarget string
	flag.StringVar(&appId, "app-id", "", "lark app id (you can also use env LARK_APP_ID)")
	flag.StringVar(&appSecret, "app-secret", "", "lark app secret (you can also use env LARK_APP_SECRET)")
	flag.StringVar(&imageType, "type", "message", "image type (message or avatar)")
	flag.StringVar(&sendTarget, "send", "", "also send image message to open_id, user_id, email or chat_id")
	flag.Usage = func() {
		fmt.Fprintf(
			flag.CommandLine.Output(),
			"Usage: %s [file ...] - %s\n", os.Args[0],
			"upload images from files or stdin to lark",
		)
		flag.PrintDefaults()
	}
	flag.Parse()

	if appId == "" {
		appId = os.Getenv("LARK_APP_ID")
	}
	if appId == "" {
		die("error: empty app id")
	}

	if appSecret == "" {
		appSecret = os.Getenv("LARK_APP_SECRET")
	}
	if appSecret == "" {
		die("error: empty app secret")
	}

	l := larkslim.API{
		AppId:     appId,
		AppSecret: appSecret,
	}

	args := flag.Args()
	var uploadFunc func(io.Reader) (string, error)
	if imageType == "message" {
		uploadFunc = l.UploadMessageImage
	} else if imageType == "avatar" {
		uploadFunc = l.UploadAvatarImage
	} else {
		die("unknown image type")
	}

	var hasErrors bool

	process := func(key string) {
		if sendTarget != "" {
			err := l.SendImageMessage(sendTarget, key)
			if err != nil {
				hasErrors = true
				fmt.Fprintln(os.Stderr, err)
			}
		}
		fmt.Println(key)
	}

	if len(args) == 0 {
		key, err := uploadFunc(os.Stdin)
		if err != nil {
			die(err)
		}
		process(key)
		return
	}

	for _, fn := range args {
		f, err := os.Open(fn)
		if err != nil {
			hasErrors = true
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		key, err := uploadFunc(f)
		f.Close()
		if err != nil {
			hasErrors = true
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		process(key)
	}
	if hasErrors {
		os.Exit(1)
	}
}

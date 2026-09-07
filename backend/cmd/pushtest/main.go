// Command pushtest sends one Web Push to a subscription and prints the push
// service's status + response body — used to diagnose 4xx from FCM/Apple.
// All inputs come from env: PUSH_ENDPOINT, PUSH_P256DH, PUSH_AUTH,
// VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, VAPID_SUBJECT.
package main

import (
	"fmt"
	"io"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	resp, err := webpush.SendNotification([]byte(`{"title":"Kabanos тест","body":"проверка доставки","url":"/dashboard"}`),
		&webpush.Subscription{
			Endpoint: os.Getenv("PUSH_ENDPOINT"),
			Keys:     webpush.Keys{P256dh: os.Getenv("PUSH_P256DH"), Auth: os.Getenv("PUSH_AUTH")},
		},
		&webpush.Options{
			Subscriber:      os.Getenv("VAPID_SUBJECT"),
			VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
			VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
			TTL:             60,
		})
	if err != nil {
		fmt.Println("send error:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("status:", resp.StatusCode)
	fmt.Println("body:", string(body))
}

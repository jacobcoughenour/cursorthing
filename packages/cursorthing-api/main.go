package main

import (
	"context"
	"net/http"
	"os"

	"github.com/centrifugal/centrifuge"
	"github.com/charmbracelet/log"
)

func auth(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Put authentication Credentials into request Context.
		// Since we don't have any session backend here we simply
		// set user ID as empty string. Users with empty ID called
		// anonymous users, in real app you should decide whether
		// anonymous users allowed to connect to your server or not.
		cred := &centrifuge.Credentials{
			UserID: "",
		}
		newCtx := centrifuge.SetCredentials(ctx, cred)
		r = r.WithContext(newCtx)
		h.ServeHTTP(w, r)
	})
}

func main() {

	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)

	node, err := centrifuge.New(centrifuge.Config{
		LogLevel: centrifuge.LogLevelDebug,
		LogHandler: func(e centrifuge.LogEntry) {
			if e.Level == centrifuge.LogLevelError {
				log.Error("[CF]", e.Message, e.Fields)
				return
			}
			if e.Level == centrifuge.LogLevelInfo {
				log.Info("[CF]", e.Message, e.Fields)
				return
			}
			if e.Level == centrifuge.LogLevelWarn {
				log.Warn("[CF]", e.Message, e.Fields)
				return
			}
			log.Debug("[CF]", e.Message, e.Fields)
		},
	})
	if err != nil {
		log.Fatal(err)
		panic(err)
	}

	node.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		return centrifuge.ConnectReply{
			Credentials: &centrifuge.Credentials{
				UserID: "",
			},
		}, nil
	})

	node.OnConnect(func(client *centrifuge.Client) {

		transportName := client.Transport().Name()
		transportProto := client.Transport().Protocol()
		log.Infof("client connected via %s (%s)", transportName, transportProto)

		client.OnSubscribe(func(e centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			log.Infof("client subscribed to channel %s", e.Channel)

			// todo we can validate the url here and if it's not normalized we reject. The other way to do it is to somehow redirect the client to the normalized url but i don't know if that is possible.

			cb(centrifuge.SubscribeReply{}, nil)
		})

		client.OnPublish(func(e centrifuge.PublishEvent, cb centrifuge.PublishCallback) {
			log.Infof("client published to channel %s", e.Channel)
			cb(centrifuge.PublishReply{}, nil)
		})

		client.OnDisconnect(func(e centrifuge.DisconnectEvent) {
			log.Info("client disconnected")
		})
	})

	// start the node
	if err := node.Run(); err != nil {
		log.Fatal(err)
	}

	// create http server

	wsHandler := centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{
		CheckOrigin: func(r *http.Request) bool {
			// allow any origin
			return true
		},
	})
	http.Handle("/connection/websocket", auth(wsHandler))

	// http.Handle("/normalize", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	// 	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 	slug := r.FormValue("url")

	// 	u, err := url.Parse(slug)
	// 	if err != nil {
	// 		http.Error(w, "invalid url", http.StatusBadRequest)
	// 		return
	// 	}

	// 	if u.Scheme != "https" {
	// 		http.Error(w, "scheme must be https", http.StatusBadRequest)
	// 		return
	// 	}
	// 	if u.Host == "" {
	// 		http.Error(w, "missing host", http.StatusBadRequest)
	// 		return
	// 	}
	// 	u.Host = strings.TrimPrefix(u.Host, "www.")

	// 	w.Write([]byte(fmt.Sprintf("%s/%s", u.Host, strings.TrimPrefix(u.Path, "/"))))
	// }))

	log.Printf("starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

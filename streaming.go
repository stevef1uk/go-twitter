package main

import (
	"bytes"
	"io/ioutil"
	"strings"
	"time"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/coreos/pkg/flagutil"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

func main() {
	flags := flag.NewFlagSet("user-auth", flag.ExitOnError)
	consumerKey := flags.String("consumer-key", "", "Twitter Consumer Key")
	consumerSecret := flags.String("consumer-secret", "", "Twitter Consumer Secret")
	accessToken := flags.String("access-token", "", "Twitter Access Token")
	accessSecret := flags.String("access-secret", "", "Twitter Access Secret")
	index_ptr := flags.Int("index", 99999, "ID to start insert from")
	apiKey := flags.String("openfaas-api-key", "", "OpenFaas Function API Key")
	flags.Parse(os.Args[1:])
	flagutil.SetFlagsFromEnv(flags, "TWITTER")
	index := *index_ptr

	if *consumerKey == "" || *consumerSecret == "" || *accessToken == "" || *accessSecret == "" {
		log.Fatal("Consumer key/secret and Access token/secret required")
	}
	if *apiKey == ""  {
		log.Fatal("OpenFaas Function API Key required")
	} else {
		fmt.Println("Api Key = ", *apiKey)
	}

	config := oauth1.NewConfig(*consumerKey, *consumerSecret)
	token := oauth1.NewToken(*accessToken, *accessSecret)
	// OAuth1 http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter Client
	client := twitter.NewClient(httpClient)

	// Convenience Demux demultiplexed stream messages
	demux := twitter.NewSwitchDemux()
	demux.Tweet = func(tweet *twitter.Tweet) {
		index = index + 1
		fmt.Println(tweet.Text)

		//url := "https://gw.sjfisher.com/async-function/stevef1uk-go1"
		url := "http://gw.sjfisher.com/async-function/go1"
		//test1 := cleanTweet( `"This" 'is a test'`)
		//_ = test1
		toInsert := `"id":` + strconv.Itoa(index) + "," + ` "message": ` + `"` + cleanTweet(tweet.Text) + `"`
		data := []byte("{" + toInsert + "}")

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
		if err != nil {
			log.Fatal("Error reading request. ", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", *apiKey)

		_, err = handlePOST(req, index)
		if err != nil {
			log.Print("ERRROR POST tweet failure! = " + err.Error())
		}
	}
	demux.DM = func(dm *twitter.DirectMessage) {
		fmt.Println(dm.SenderID)
	}
	demux.Event = func(event *twitter.Event) {
		fmt.Printf("%#v\n", event)
	}

	fmt.Println("Starting Stream...")

	// FILTER
	filterParams := &twitter.StreamFilterParams{
		Track:         []string{"virus"},
		Language:      []string{"en"},
		StallWarnings: twitter.Bool(true),
	}
	stream, err := client.Streams.Filter(filterParams)
	if err != nil {
		log.Fatal(err)
	}

	// USER (quick test: auth'd user likes a tweet -> event)
	// userParams := &twitter.StreamUserParams{
	// 	StallWarnings: twitter.Bool(true),
	// 	With:          "followings",
	// 	Language:      []string{"en"},
	// }
	// stream, err := client.Streams.User(userParams)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// SAMPLE
	// sampleParams := &twitter.StreamSampleParams{
	// 	StallWarnings: twitter.Bool(true),
	// }
	// stream, err := client.Streams.Sample(sampleParams)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Receive messages until stopped or stream quits
	go demux.HandleChan(stream.Messages)

	// Wait for SIGINT and SIGTERM (HIT CTRL-C)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	fmt.Println("Stopping Stream...")
	stream.Stop()
}

func handlePOST(req *http.Request, index int) (string, error) {
	ret := ""

	log.Println("In HandlePOST, Host = " + req.Host + "index = " + strconv.Itoa(index))
	//log.Println("Body = " + string(req.Body))

	client := &http.Client{Timeout: time.Second * 5}

	// Validate cookie and headers are attached
	//fmt.Println(req.Cookies())
	//fmt.Println(req.Header)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error reading response. ", err)
	}
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	//fmt.Println("response Headers:", resp.Header)

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading body. ", err)
	}
	//fmt.Printf("%s\n", body)
	return ret, err
}

func cleanTweet(data string) string {

	res1 := strings.ReplaceAll(data, `"`, "")
	res2 := strings.ReplaceAll(res1, `'`, "")
	res1 = strings.ReplaceAll(res2, "`", "")

	return res1
}

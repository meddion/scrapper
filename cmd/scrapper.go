package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/meddion/scrapper/internal/db"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/cobra"
)

const (
	DEFAULT_SEARCH   = "https://www.google.com/search?q="
	WORKER_POOL_SIZE = 10
)

var scrapperCmd = &cobra.Command{
	Use:   "scrapper",
	Short: "Scrapper gets the links from a page",
	Args:  cobra.ExactValidArgs(1),
	Run:   runScrapper,
}

func runScrapper(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go OsSignalHandler(cancel)

	db := db.GetKafkaConn()

	addr := DEFAULT_SEARCH + args[0]
	res := scrapperProducer(ctx, addr)

	var wg sync.WaitGroup
	wg.Add(WORKER_POOL_SIZE)
	for i := 0; i < WORKER_POOL_SIZE; i++ {
		go func() {
			defer wg.Done()

			scrapperConsumer(ctx, db, res)
		}()
	}

	wg.Wait()
	<-res
}

type resp struct {
	url string
	err error
}

func scrapperProducer(ctx context.Context, addr string) <-chan resp {
	res := make(chan resp)

	go func() {
		defer close(res)

		doc, err := getDoc(ctx, addr)
		if err != nil {
			select {
			case <-ctx.Done():
			case res <- resp{err: err}:
			}
			return
		}

		filter := func(i int, s *goquery.Selection) bool {
			_, ok := s.Attr("href")
			return i >= 16 && i < 26 && ok
		}

		doc.Find("#main a").FilterFunction(filter).Each(func(i int, s *goquery.Selection) {
			var (
				link string
				err  error
			)
			link, _ = s.Attr("href")
			link = strings.Split(link, "&sa=")[0]
			link = strings.Trim(link, "/url?q=")
			link, err = url.QueryUnescape(link)

			select {
			case <-ctx.Done():
			case res <- resp{url: link, err: err}:
			}
		})
	}()

	return res
}

func scrapperConsumer(ctx context.Context, db *kafka.Conn, res <-chan resp) {
	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-res:
			if !ok {
				return
			}
			if r.err != nil {
				log.Printf("got from producer: %s", r.err.Error())
				continue
			}

			// TODO: add deadline of sort
			// Maybe: kafkaConn.SetWriteDeadline(time.Now().Add(time.Second * 3))
			if _, err := db.Write([]byte(r.url)); err != nil {
				log.Printf("on writing to kafka from a worker: %s", err.Error())
			}

			log.Printf("Added %s to Kafka :)", r.url)
		}
	}
}

func getDoc(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

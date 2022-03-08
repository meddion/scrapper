package cmd

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/meddion/scrapper/internal/db"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/cobra"
)

var fetcherCmd = &cobra.Command{
	Use:   "fetcher",
	Short: "fetches data from kafka and does something w/ it",
	Args:  cobra.ExactValidArgs(0),
	Run:   runFetcher,
}

const MAX_READ_PER_EPOCH = 10

func runFetcher(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go OsSignalHandler(cancel)

	db := db.GetKafkaConn()

	res := fetcherProducer(ctx, db)

	var wg sync.WaitGroup
	wg.Add(WORKER_POOL_SIZE)
	for i := 0; i < WORKER_POOL_SIZE; i++ {
		go func() {
			defer wg.Done()

			fetcherConsumer(ctx, res)
		}()
	}

	wg.Wait()
	<-res
}

func fetcherProducer(ctx context.Context, db *kafka.Conn) <-chan resp {
	res := make(chan resp)

	go func() {
		defer close(res)

		db.SetReadDeadline(time.Now().Add(time.Second * 3))
		batch := db.ReadBatch(10e3, 1e6)
		b := make([]byte, 10e3)
	L:
		for i := 0; i < MAX_READ_PER_EPOCH; i++ {
			n, err := batch.Read(b)
			if err != nil {
				break
			}

			select {
			case <-ctx.Done():
				break L
			case res <- resp{url: string(b[:n])}:
			}

		}

		if err := batch.Close(); err != nil {
			if err, ok := err.(kafka.Error); ok && err == kafka.RequestTimedOut {
				return
			}

			select {
			case <-ctx.Done():
			case res <- resp{err: err}:
			}
		}
	}()

	return res
}

func fetcherConsumer(ctx context.Context, res <-chan resp) {
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

			doc, err := getDoc(ctx, r.url)
			if err != nil {
				log.Printf("on fetching from %s: %s", r.url, err.Error())
				continue
			}

			title := doc.Find("head title").Text()
			if title != "" {
				log.Printf("Title: %s", title)
			}
		}
	}
}

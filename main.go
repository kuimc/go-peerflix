package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

// Exit statuses.
const (
	_ = iota
	exitNoTorrentProvided
	exitErrorInClient
)

func main() {
	// Parse flags.
	var port int
	var vlc *bool
	var mplayer *bool
	var seed *bool

	vlc = flag.Bool("vlc", false, "Open vlc to play the file")
	mplayer = flag.Bool("mplayer", false, "Open mplayer to play the file")
	flag.IntVar(&port, "port", 8080, "Port to stream the video on")
	seed = flag.Bool("seed", false, "Seed after finished downloading")
	flag.Parse()
	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(exitNoTorrentProvided)
	}

	// Start up the torrent client.
	client, err := NewClient(flag.Arg(0), port, *seed)
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(exitErrorInClient)
	}

	// Http handler.
	go func() {
		http.HandleFunc("/", client.GetFile)
		log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), nil))
	}()

	// Open vlc to play.
	if *vlc {
		go func() {
			for !client.ReadyForPlayback() {
				time.Sleep(time.Second)
			}
			playInVlc(port)
		}()
	}
	if *mplayer {
		go func() {
			for !client.ReadyForPlayback() {
				time.Sleep(time.Second)
			}
			playInMplayer(port)
		}()
	}

	// Handle exit signals.
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func(interruptChannel chan os.Signal) {
		for range interruptChannel {
			log.Println("Exiting...")
			client.Close()
			os.Exit(0)
		}
	}(interruptChannel)

	// Cli render loop.
	for {
		client.Render()
		time.Sleep(time.Second)
	}
}

func playInVlc(port int) {
	log.Printf("Playing in vlc")

	command := []string{"vlc"}
	if runtime.GOOS == "darwin" {
		command = []string{"open", "-a", "vlc"}
	}
	command = append(command, "http://localhost:"+strconv.Itoa(port))

	if err := exec.Command(command[0], command[1:]...).Start(); err != nil {
		log.Printf("Error opening vlc: %s\n", err)
	}
}

func playInMplayer(port int) {
		log.Printf("Playing in mplayer")
		command := []string{"mplayer", "http://localhost:"+strconv.Itoa(port), "-cache 2048"}
		err := exec.Command(command[0], command[1]).Start()

		if err != nil {
			log.Printf("Error opening mplayer: %s\n", err)
		}
}

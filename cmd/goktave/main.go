package main

import (
	"fmt"
	"time"
	"bufio"
	"os"

	"github.com/Kush-Singh-26/goktave/internal/extractor"
	"github.com/Kush-Singh-26/goktave/internal/player"
	"github.com/Kush-Singh-26/goktave/internal/provider"
)

func main() {
	fmt.Println("1. Initializing Audio System...")
	speaker, err := player.NewSpeaker()
	if err != nil {
		panic(err)
	}
	pl := player.New(speaker)

	fmt.Println("enter query")
	reader := bufio.NewReader(os.Stdin)
	query, _ := reader.ReadString('\n')
	fmt.Println("2. Searching YouTube Music...")
	prov := provider.NewYTMusic()
	results, err := prov.Search(query)
	if err != nil || len(results) == 0 {
		fmt.Printf("Search failed: %v\n", err)
		return
	}

	target := results[0]
	fmt.Printf("   Found: %s by %s\n", target.Title, target.Artist)

	fmt.Println("3. Extracting Stream URL...")
	ext := extractor.New()
	info, err := ext.Extract(target.VideoID)
	if err != nil {
		fmt.Printf("Extraction failed: %v\n", err)
		return
	}

	fmt.Println("4. Playing Audio...")
	err = pl.Play(info.URL)
	if err != nil {
		fmt.Printf("Playback failed: %v\n", err)
		return
	}

	fmt.Println("Playing for 15 seconds... (listen closely!)")
	time.Sleep(15 * time.Second)
	
	pl.Stop()
	fmt.Println("Playback stopped cleanly.")
}
// See the accompanying README for more information.

package main

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"git.sr.ht/~adnano/go-gemini"
	"github.com/qpliu/qrencode-go/qrencode"
	"github.com/t-900-a/rss"
)

const VERSION = "1.0"

// ---------------------------------------------------------------------------
// Local error object that conforms to the Go error interface.
// ---------------------------------------------------------------------------

type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// ---------------------------------------------------------------------------
// Local functions
// ---------------------------------------------------------------------------

// Convenience function to print a message (printf style) to standard error
// and exit with a non-zero exit code.
func die(format string, args ...interface{}) {
	os.Stderr.WriteString(fmt.Sprintf(format, args...) + "\n")
	os.Exit(1)
}

func getEmoji() string {
	var buf bytes.Buffer
	rand.Seed(time.Now().UTC().UnixNano())
	i := rand.Int() % len(emojis)
	buf.WriteRune(emojis[i])
	return buf.String()
}

// Parse the command line arguments. For now, this is simple, because this
// program requires very few arguments. If something more complicated is
// needed, consider the Go "flag" module or github.com/docopt/docopt-go
func parseArgs() (string, string, string, string, string, error) {
	prog := path.Base(os.Args[0])
	usage := fmt.Sprintf(`%s, version %s

Usage:
  %s [payment uri]
  %s [tx hash]
  %s -h|--help
`, prog, VERSION, prog, prog)
	// TODO REDO variable handling
	var websiteUri string
	var streamUri string
	var txHash string
	var pmtUri string
	var pmtViewKey string
	var err error
	switch len(os.Args) {
	case 2:
		websiteUri = os.Args[1]
	case 3:
		websiteUri = os.Args[1]
		streamUri = os.Args[2]
	case 4:
		websiteUri = os.Args[1]
		streamUri = os.Args[2]
		txHash = os.Args[3]
	case 5:
		websiteUri = os.Args[1]
		streamUri = os.Args[2]
		txHash = os.Args[3]
		pmtUri = os.Args[4]
	case 6:
		websiteUri = os.Args[1]
		streamUri = os.Args[2]
		txHash = os.Args[3]
		pmtUri = os.Args[4]
		pmtViewKey = os.Args[5]
	case 7:
		{
			if (os.Args[1] == "-h") || (os.Args[1] == "--help") {
				err = &Error{usage}
			}
		}
	default:
		err = &Error{usage}
	}

	return websiteUri, streamUri,txHash, pmtUri, pmtViewKey, err
}

// ---------------------------------------------------------------------------
// Main program
// ---------------------------------------------------------------------------

func main() {
	ti := time.Now()
	websiteUri, streamUri, txHash, pmtUri, pmtViewKey, err := parseArgs()
	if err != nil {
		die(err.Error())
	}

	// start the radio
	cmd := exec.Command("mpc", "play")
	err = cmd.Run()
	if err != nil {
		die(err.Error())
	}

	if len(txHash) > 0 {
		// tx-notify runs 3 times per transaction, we only want to run the bot 1 time per transaction
		file, err := os.OpenFile("lastrun.txt", os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		b, err := ioutil.ReadAll(file)
		if string(b) == txHash {
			log.Fatal("TX already processed")
			os.Exit(0)
		} else {
			_, err := file.WriteAt([]byte(txHash), 0) // Write at 0 beginning
			if err != nil {
				log.Fatalf("failed writing to file: %s", err)
			}
		}
	}

	// write jukebox status to gemini text
	var t gemini.Text
	title := getEmoji() + " Coin Inserted " + getEmoji()
	heading := gemini.LineHeading1(title)
	t = append(t, heading)

	newLine := gemini.LineText("\n")
	t = append(t, newLine)

	if len(txHash) > 0 {
		link := "https://xmrchain.net/tx/" + txHash
		txt := "Music requested, associated transaction found here"
		blockExplorer := &gemini.LineLink{URL: link, Name: txt}
		t = append(t, blockExplorer)
	}

	newLine = gemini.LineText("\n")
	t = append(t, newLine)
	body := "```" + botsay("Jukebox Radio now playing") + "```"
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		fortuneLine := gemini.LineText(scanner.Text())
		t = append(t, fortuneLine)
	}

	dateLine := gemini.LineQuote(ti.Format("2006-01-02T15:04:05Z"))
	t = append(t, dateLine)

	newLine = gemini.LineText("\n")
	t = append(t, newLine)

	if len(pmtUri) > 0 {

		txt := "Tune in here"
		listenUri := &gemini.LineLink{URL: streamUri, Name: txt}
		t = append(t, listenUri)

		// generate qr code to scan
		// only works for gemini cli clients

		newLine = gemini.LineText("\n")
		t = append(t, newLine)

		txt = "Music stopped? Send any amount of Monero to get it playing again."
		pmtAddress := &gemini.LineLink{URL: pmtUri, Name: txt}
		t = append(t, pmtAddress)
		// generate qr code to scan
		// only works for gemini cli clients

		newLine = gemini.LineText("\n")
		t = append(t, newLine)

		grid, err := qrencode.Encode(pmtUri, qrencode.ECLevelL)
		if err != nil {
			log.Fatal(err)
		}
		var b bytes.Buffer
		grid.TerminalOutput(&b)
		qrCode := "```" + b.String() + "```"
		scanner := bufio.NewScanner(strings.NewReader(qrCode))
		for scanner.Scan() {
			fortuneLine := gemini.LineText(scanner.Text())
			t = append(t, fortuneLine)
		}

		newLine = gemini.LineText("\n")
		t = append(t, newLine)

		txt = "Don't have any Monero? Learn more here"
		link := "https://getmonero.org/"
		info := &gemini.LineLink{URL: link, Name: txt}
		t = append(t, info)
	}

	fileName := "jukebox_" + ti.Format("2006_01_02_15_04_05") + ".gmi"
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	for _, line := range t {
		file.WriteString(line.String() + "\n")
	}

	// write ATOM FEED
	// TODO read and append to existing feed if there
	// payment data
	var pmtData []rss.AtomLink
	pmtLink := rss.AtomLink{
		Href: pmtUri,
		Rel:  "payment",
		Type: "application/monero-paymentrequest",
	}
	pmtData = append(pmtData, pmtLink)
	pmtLink = rss.AtomLink{
		Href: pmtViewKey,
		Rel:  "payment",
		Type: "application/monero-viewkey",
	}
	pmtData = append(pmtData, pmtLink)
	// feed website
	var websiteLink []rss.AtomLink
	wbLink := rss.AtomLink{
		Href: websiteUri,
	}
	websiteLink = append(websiteLink, wbLink)
	// this item
	var itemLink []rss.AtomLink
	thisLink := websiteUri + "/gemlog/" + fileName
	itmLink := rss.AtomLink{
		Href: thisLink,
	}
	itemLink = append(itemLink, itmLink)
	// feed items
	var feedItems []rss.AtomItem
	feedItem := rss.AtomItem{
		Title:   title,
		Content: rss.RAWContent{RAWContent: "More Coins More Music, ya dig"},
		Links:   itemLink,
		Date:    ti.Format("2006-01-02T15:04:05Z"),
		ID:      fileName,
	}
	feedItems = append(feedItems, feedItem)

	feed := &rss.AtomFeed{
		Title:       "Jukebox Bot",
		Description: "Fine melodies in the Gemini Space",
		Author: rss.AtomAuthor{
			Name:       "Anon",
			URI:        websiteUri,
			Extensions: pmtData,
		},
		Link:    websiteLink,
		Items:   feedItems,
		Updated: ti.Format("2006-01-02T15:04:05Z"),
	}
	atomFile, err := xml.MarshalIndent(feed, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("atom.xml", atomFile, 0644)
	if err != nil {
		log.Fatal(err)
	}

	// write entry to the gemlog
	// assumes the script is ran within the /gemlog/ directory
	gemlog, err := os.OpenFile("index.gmi", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer gemlog.Close()

	var offset int64 = 0
	var whence int = 2 // end of file
	newPosition, err := gemlog.Seek(offset, whence)
	if err != nil {
		log.Fatal(err)
	}

	_, err = gemlog.WriteAt([]byte("\n=> "+fileName+" "+title), newPosition) // Write at end
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}
}

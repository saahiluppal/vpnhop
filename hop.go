package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Status struct {
	Connected string
	Country   string
	City      string
	Uptime    time.Duration
}

func trim_useless(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsNumber(r)
}

func nordvpn_status(channel chan Status, wg *sync.WaitGroup) {

	// Populating Status Struct
	var output bytes.Buffer

	nv_status := exec.Command("nordvpn", "status")
	nv_status.Stdout = &output

	err := nv_status.Run()
	if err != nil {
		log.Fatal(err)
	}

	status_out := fmt.Sprintf("%s", output.String())
	output.Reset()

	var status Status

	for idx, val := range strings.Split(status_out, "\n") {
		val = strings.TrimFunc(val, trim_useless)

		if idx == 0 {
			if strings.HasSuffix(val, "Connected") {
				status.Connected = "Connected"
			} else if strings.HasSuffix(val, "Disconnected") {
				status.Connected = "Disconnected"
			} else if strings.HasSuffix(val, "Reconnecting") {
				status.Connected = "Reconnecting"
			} else {
				log.Fatal("Something in line 'Status: Connected' has changed")
			}
		}

		if status.Connected == "Connected" {
			if strings.HasPrefix(val, "Country") {
				country := strings.Split(val, " ")
				status.Country = strings.Join(country[1:], " ")
			}
			if strings.HasPrefix(val, "City") {
				city := strings.Split(val, " ")
				status.City = strings.Join(city[1:], " ")
			}
			if strings.HasPrefix(val, "Uptime") {
				uptime := strings.Split(val, " ")
				var second int
				var minute int
				var hour int
				if len(uptime) == 3 {
					second, _ = strconv.Atoi(uptime[len(uptime)-2])
				} else if len(uptime) == 5 {
					second, _ = strconv.Atoi(uptime[len(uptime)-2])
					minute, _ = strconv.Atoi(uptime[len(uptime)-4])
				} else if len(uptime) == 7 {
					second, _ = strconv.Atoi(uptime[len(uptime)-2])
					minute, _ = strconv.Atoi(uptime[len(uptime)-4])
					hour, _ = strconv.Atoi(uptime[len(uptime)-6])
				}
				status.Uptime = time.Second * time.Duration(
					(3600*hour)+(60*minute)+(second))
			}
		}
	}

	channel <- status
	// Popluating Status Struct ends

}

func nordvpn_connect(country string) (bool, error) {
	nv_connect := exec.Command("nordvpn", "connect", country)

	err := nv_connect.Run()
	if err != nil {
		return false, err
	} else {
		return true, nil
	}

}

func nordvpn_countries(channel chan []string, wg *sync.WaitGroup) {

	// Populating Countries
	var output bytes.Buffer
	var countries []string

	nv_countries := exec.Command("nordvpn", "countries")
	nv_countries.Stdout = &output

	err := nv_countries.Run()
	if err != nil {
		log.Fatal(err)
	}

	countries_out := fmt.Sprintf("%s", output.String())
	output.Reset()

	for _, country := range strings.Fields(countries_out) {
		country = strings.TrimFunc(country, trim_useless)
		if len(country) == 0 {
			continue
		}
		countries = append(countries, country)
	}

	channel <- countries
	// Populating Countries ends

}

func is_internet_working(channel chan bool, wg *sync.WaitGroup) {

	// Checking internet connection
	var internet bool

	_, err := http.Get("https://www.google.com/")
	if err != nil {
		internet = false
	} else {
		internet = true
	}

	channel <- internet
	// Checking internet connection ends

}

func info() (Status, []string, bool) {

	wg := new(sync.WaitGroup)
	internet_channel := make(chan bool)
	countries_channel := make(chan []string)
	status_channel := make(chan Status)

	// First goroutine should be internet_working because it takes most time
	go is_internet_working(internet_channel, wg)
	go nordvpn_status(status_channel, wg)
	go nordvpn_countries(countries_channel, wg)

	return <-status_channel, <-countries_channel, <-internet_channel

}

const MIN_UPTIME = 10 * time.Minute
const HOP_AFTER = 1 * time.Hour

func main() {

	if HOP_AFTER <= MIN_UPTIME {
		panic(fmt.Sprintf("Minimum Uptime required is", MIN_UPTIME,
			", But you are trying to hop after", HOP_AFTER))
	}

	status, countries, internet := info()
	if status.Connected == "Connected" {

		if internet {
			if MIN_UPTIME <= status.Uptime {
				rand.Seed(time.Now().Unix())
				_, err := nordvpn_connect(countries[rand.Intn(len(countries))])
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Println("Already Connected to", status.Country)
				fmt.Println("Minimum Uptime =", MIN_UPTIME, ", Server Uptime =", status.Uptime)
			}
		} else {
			log.Fatal("Connected to VPN but there's no internet connection")
		}

	} else if status.Connected == "Disconnected" {

		if internet {
			rand.Seed(time.Now().Unix())
			_, err := nordvpn_connect(countries[rand.Intn(len(countries))])
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal("Internet not available")
		}

	} else if status.Connected == "Reconnecting" {
		log.Fatal("It's Reconnecting, Nordvpn might be frozen")
	}

	ticker := time.NewTicker(HOP_AFTER)

	for range ticker.C {

		fmt.Println("Starting Service")
		status, countries, internet := info()
		if status.Connected == "Connected" {

			if internet {
				if MIN_UPTIME <= status.Uptime {
					rand.Seed(time.Now().Unix())
					_, err := nordvpn_connect(countries[rand.Intn(len(countries))])
					if err != nil {
						log.Fatal(err)
					}
				} else {
					fmt.Println("Already Connected to", status.Country)
					fmt.Println("Minimum Uptime =", MIN_UPTIME, ", Server Uptime =", status.Uptime)
				}
			} else {
				log.Fatal("Connected to VPN but there's no internet connection")
			}

		} else if status.Connected == "Disconnected" {

			if internet {
				rand.Seed(time.Now().Unix())
				_, err := nordvpn_connect(countries[rand.Intn(len(countries))])
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal("Internet not available")
			}

		} else if status.Connected == "Reconnecting" {
			log.Fatal("It's Reconnecting, Nordvpn might be frozen")
		}
		fmt.Println("Stopping Service")

	}

}

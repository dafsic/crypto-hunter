package main

import (
	"fmt"

	"github.com/dafsic/crypto-hunter/app"
	"github.com/dafsic/crypto-hunter/kraken"
	"github.com/dafsic/crypto-hunter/kraken_grid/bot"
	"github.com/dafsic/crypto-hunter/kraken_grid/dao"
	"github.com/dafsic/crypto-hunter/log"
)

func main() {
	app := app.NewApplication("CryptoHunter", "A trading bot for kraken exchange")
	app.Install(
		&log.Module{},
		&kraken.Module{},
		&dao.Module{},
		&bot.Module{},
	)
	if err := app.Run(); err != nil {
		fmt.Println(err)
		return
	}
}

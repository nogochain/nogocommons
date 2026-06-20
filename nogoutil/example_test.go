package nogoutil_test

import (
	"fmt"
	"math"

	"github.com/nogochain/nogocommons/nogoutil"
)

func ExampleAmount() {

	a := nogoutil.Amount(0)
	fmt.Println("Zero Satoshi:", a)

	a = nogoutil.Amount(1e8)
	fmt.Println("100,000,000 Satoshis:", a)

	a = nogoutil.Amount(1e5)
	fmt.Println("100,000 Satoshis:", a)
	// Output:
	// Zero Satoshi: 0 BTC
	// 100,000,000 Satoshis: 1 BTC
	// 100,000 Satoshis: 0.00100000 BTC
}

func ExampleNewAmount() {
	amountOne, err := nogoutil.NewAmount(1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountOne) //Output 1

	amountFraction, err := nogoutil.NewAmount(0.01234567)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountFraction) //Output 2

	amountZero, err := nogoutil.NewAmount(0)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountZero) //Output 3

	amountNaN, err := nogoutil.NewAmount(math.NaN())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(amountNaN) //Output 4

	// Output: 1 BTC
	// 0.01234567 BTC
	// 0 BTC
	// invalid bitcoin amount
}

func ExampleAmount_unitConversions() {
	amount := nogoutil.Amount(44433322211100)

	fmt.Println("Satoshi to kBTC:", amount.Format(nogoutil.AmountKiloBTC))
	fmt.Println("Satoshi to BTC:", amount)
	fmt.Println("Satoshi to MilliBTC:", amount.Format(nogoutil.AmountMilliBTC))
	fmt.Println("Satoshi to MicroBTC:", amount.Format(nogoutil.AmountMicroBTC))
	fmt.Println("Satoshi to Satoshi:", amount.Format(nogoutil.AmountSatoshi))

	// Output:
	// Satoshi to kBTC: 444.333222111 kBTC
	// Satoshi to BTC: 444333.22211100 BTC
	// Satoshi to MilliBTC: 444333222.111 mBTC
	// Satoshi to MicroBTC: 444333222111 μBTC
	// Satoshi to Satoshi: 44433322211100 Satoshi
}

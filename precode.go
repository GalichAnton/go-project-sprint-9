package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Generator генерирует последовательность чисел 1,2,3 и т.д. и
// отправляет их в канал ch. При этом после записи в канал для каждого числа
// вызывается функция fn. Она служит для подсчёта количества и суммы
// сгенерированных чисел.
func Generator(ctx context.Context, ch chan<- int64, fn func(int64)) {
	defer close(ch)
	var n int64 = 1
	for {
		select {
		case <-ctx.Done():
			return
		default:
			ch <- n
			fn(n)
			n = n + 1
		}

	}
}

// Worker читает число из канала in и пишет его в канал out.
func Worker(in <-chan int64, out chan<- int64) {
	defer close(out)

	for {
		v, ok := <-in
		if !ok {
			return
		}

		out <- v

		time.Sleep(1 * time.Millisecond)
	}

}

func main() {
	chIn := make(chan int64)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var inputSum int64
	var inputCount int64

	go Generator(
		ctx, chIn, func(i int64) {
			atomic.AddInt64(&inputSum, i)
			atomic.AddInt64(&inputCount, 1)
		},
	)

	const NumOut = 10
	outs := make([]chan int64, NumOut)
	for i := 0; i < NumOut; i++ {
		outs[i] = make(chan int64)
		go Worker(chIn, outs[i])
	}

	amounts := make([]int64, NumOut)
	chOut := make(chan int64, NumOut)

	var wg sync.WaitGroup

	for i, outCh := range outs {
		wg.Add(1)
		go func(in <-chan int64, idx int) {
			defer wg.Done()
			for num := range in {
				amounts[idx]++
				chOut <- num
			}
			return
		}(outCh, i)
	}

	go func() {
		wg.Wait()
		close(chOut)
	}()

	var count int64
	var sum int64

	for val := range chOut {
		count++
		sum += val
	}

	fmt.Println("Количество чисел", inputCount, count)
	fmt.Println("Сумма чисел", inputSum, sum)
	fmt.Println("Разбивка по каналам", amounts)

	if inputSum != sum {
		log.Fatalf("Ошибка: суммы чисел не равны: %d != %d\n", inputSum, sum)
	}
	if inputCount != count {
		log.Fatalf("Ошибка: количество чисел не равно: %d != %d\n", inputCount, count)
	}
	for _, v := range amounts {
		inputCount -= v
	}
	if inputCount != 0 {
		log.Fatalf("Ошибка: разделение чисел по каналам неверное\n")
	}
}

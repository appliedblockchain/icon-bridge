package algo

/* import (
	"testing"
)

const (
	feeGathering = "0xf8d0b8396274703a2f2f3078322e69636f6e2f637833666263643763323562653961616336336264393434613062386434306265636461613335343235b8396274703a2f2f307836312e6273632f30783835353138334165664441393843643130436633313438386532366646353939363835363235613783626d6300b853f8518c466565476174686572696e67b842f840b8396274703a2f2f3078322e69636f6e2f687832373531333661633936396161646135656466623534333939313264623564636431303734626466c483627473"
	coinTransfer = "0xf8fbb8396274703a2f2f3078312e69636f6e2f637832336139316565336464323930343836613931313361366134323432393832356438313364653533b83b6274703a2f2f30783232382e736e6f772f3078663830384662623535434644446133374430353142393632623739313861384433393831666544358362747382024cb87af87800b875f873aa687835666135336432396230303539646562316166303366386564623964623130396636343936373966aa307835433039463435453944316563334441653632323431383862613730613566363631393745443139dcdb906274702d3078312e69636f6e2d4943588953401e65833f620000"
)

func Test_DecodeRelayMessage(t *testing.T) {
	svcName, svcArgs, err := DecodeRelayMessage(feeGathering)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if svcName != "FeeGathering" {
		t.Error("Wrong service name, expected 'FeeGathering', got %w", svcName)
	}
	if _, ok := svcArgs.(FeeGatheringSvc); !ok {
		t.Error("Wrong args type for FeeGathering:", svcArgs)
	}

	svcName, svcArgs, err = DecodeRelayMessage(coinTransfer)
	if err != nil {
		t.Error(err)
	}
	if svcName != "CoinTransfer" {
		t.Error("Wrong service name, expected 'CoinTransfer', got %w", svcName)
	}
	if _, ok := svcArgs.(CoinTransferSvc); !ok {
		t.Error("Wrong args type for CoinTransfer:", svcArgs)
	}
} */

// func Test_Abi(t *testing.T) {
// 	s, err := createTestSender(testnetAccess)
// 	if err != nil {
// 		t.Logf("Failed creting new sender:%v", err)
// 		t.FailNow()
// 	}
// 	ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)

// 	abiCall, err := s.(*sender).callAbi(ctx, AbiFunc{"sendMessage",
// 		[]interface{}{"this", "string", "hhll", []byte{0x01, 0x02, 0x03}}})

// 	if err != nil {
// 		t.Logf("Failed to call abi:%v", err)
// 		t.FailNow()
// 	}
// 	fmt.Println(abiCall)
// }

// func Test_Segment(t *testing.T) {
// 	s, err := createTestSender(testnetAccess)
// 	if err != nil {
// 		t.Logf("Failed creting new sender:%v", err)
// 		t.FailNow()
// 	}
// 	ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)

// 	msg := &chain.Message{
// 		From: chain.BTPAddress(iconBmc),
// 		Receipts: []*chain.Receipt{{
// 			Index:  10,
// 			Height: 1,
// 			Events: []*chain.Event{
// 				{Next: "algobmc", Sequence: 19, Message: []byte{97, 98, 99, 100, 101, 102}},
// 			}}, {
// 			Index:  20,
// 			Height: 1,
// 			Events: []*chain.Event{
// 				{Next: "algobmc", Sequence: 20, Message: []byte{64, 2, 4, 111, 55, 23}},
// 			}},
// 		},
// 	}
// 	_, _, err = s.Segment(ctx, msg)

// 	if err != nil {
// 		t.Logf("Couldn't segment message:%v", err)
// 		t.FailNow()
// 	}
// }

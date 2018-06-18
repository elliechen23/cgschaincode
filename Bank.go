package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

const BankObjectType string = "Bank"
const AdminBankID string = "CBC"

type Bank struct {
	ObjectType   string      `json:"docType"`      // default set to "Bank"
	BankID       string      `json:"BankID"`       // BANK002,BANK004,BANK005,BANKCBC
	BankName     string      `json:"BankName"`     // BANK 002, BANK 004,BANK 005, BANK CBC
	BankCode     string      `json:"BankCode"`     // BankCode (002,004,005,999)
	BankTotals   []BankTotal `json:"BankTotals"`   //清算銀行總計數
	BankAccounts []string    `json:"BankAccounts"` //清算銀行下客戶帳號
}

type BankTotal struct {
	SecurityID   string `json:"SecurityID"`   //公債代號
	TotalBalance int64  `json:"TotalBalance"` //總券數
	TotalAmount  int64  `json:"TotalAmount"`  //總款數
	CreateTime   string `json:"CreateTime"`
	UpdateTime   string `json:"UpdateTime"`
}

/*
peer chaincode invoke -n mycc1 -c '{"Args":["initBank", "BANK002" , "BANK 002" , "002" ]}' -C myc
peer chaincode invoke -n mycc1 -c '{"Args":["initBank", "BANK004" , "BANK 004" , "004" ]}' -C myc
peer chaincode invoke -n mycc1 -c '{"Args":["initBank", "BANKCBC" , "BANK CBC" , "CBC" ]}' -C myc

*/
func (s *SmartContract) initBank(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	// BankID, BankName, Status
	err := checkArgArrayLength(args, 3)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("BankID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("BankName must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("BankCode must be a non-empty string")
	}

	BankID := args[0]
	BankName := strings.ToUpper(args[1])
	BankCode := strings.ToUpper(args[2])

	BankAsBytes, err := stub.GetState(BankID)
	if err != nil {
		return shim.Error(err.Error())
	} else if BankAsBytes != nil {
		errMsg := fmt.Sprintf(
			"Error: This Bank already exists (%s)",
			BankID)
		return shim.Error(errMsg)
	}

	Bank := Bank{}
	Bank.ObjectType = BankObjectType
	Bank.BankID = BankID
	Bank.BankName = BankName
	Bank.BankCode = BankCode

	BankAsBytes, err = json.Marshal(Bank)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(BankID, BankAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(BankAsBytes)
}

//peer chaincode invoke -n mycc -c '{"Args":["updateBank", "001" , "BANK001" , "1" ]}' -C myc
func (s *SmartContract) updateBank(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	// BankID, BankName, Status
	err := checkArgArrayLength(args, 3)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("BankID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("BankName must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("BankCode must be a non-empty string")
	}
	BankID := args[0]
	BankName := args[1]
	BankCode := args[2]

	BankAsBytes, err := stub.GetState(BankID)
	if err != nil {
		return shim.Error(err.Error())
	} else if BankAsBytes == nil {
		errMsg := fmt.Sprintf(
			"Error: This Bank does not exist (%s)",
			BankID)
		return shim.Error(errMsg)
	}

	Bank := Bank{}
	Bank.ObjectType = BankObjectType
	Bank.BankID = BankID
	Bank.BankName = BankName
	Bank.BankCode = BankCode

	BankAsBytes, err = json.Marshal(Bank)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(BankID, BankAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

func (s *SmartContract) deleteBank(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	err := checkArgArrayLength(args, 1)
	if err != nil {
		return shim.Error(err.Error())
	}

	BankID := args[0]

	// Access Control
	//err = verifyIdentity(stub, regulatorName)
	//if err != nil {
	//	return shim.Error(err.Error())
	//}

	valAsbytes, err := stub.GetState(BankID)
	if err != nil {
		errMsg := fmt.Sprintf(
			"Error: Failed to get state for Bank (%s)",
			BankID)
		return shim.Error(errMsg)
	} else if valAsbytes == nil {
		errMsg := fmt.Sprintf(
			"Error: Bank does not exist (%s)",
			BankID)
		return shim.Error(errMsg)
	}

	err = stub.DelState(BankID)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

//peer chaincode query -n mycc -c '{"Args":["verifyBankList", "CBC"]}' -C myc
func (s *SmartContract) verifyBankList(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	err := checkArgArrayLength(args, 1)
	if err != nil {
		return shim.Error(err.Error())
	}

	BankID := args[0]

	valAsbytes, err := stub.GetState(BankID)
	if err != nil {
		errMsg := fmt.Sprintf(
			"Error: Failed to get state for BankID (%s)",
			BankID)
		return shim.Error(errMsg)
	} else if valAsbytes == nil {
		errMsg := fmt.Sprintf(
			"Error: BankID does not exist (%s)",
			BankID)
		return shim.Error(errMsg)
	}

	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"BankID\":")
	buffer.WriteString("\"")
	buffer.WriteString(BankID)
	buffer.WriteString("\"")
	buffer.WriteString(", \"Value\":")
	buffer.WriteString(string(valAsbytes))
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())

}

func verifyIdentity(
	stub shim.ChaincodeStubInterface,
	bankID string) string {

	valAsbytes, err := stub.GetState(bankID)
	if err != nil {
		errMsg := fmt.Sprintf(
			"Error: Failed to get state for BankID (%s)",
			bankID)
		return errMsg
	} else if valAsbytes == nil {
		errMsg := fmt.Sprintf(
			"Error: BankID does not exist (%s)",
			bankID)
		return errMsg
	}

	return ""
}

//peer chaincode query -n mycc -c '{"Args":["queryAllBanks", "000" , "ZZZ"]}' -C myc
func (s *SmartContract) queryAllBanks(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}
	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := APIstub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as/is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc -c '{"Args":["getHistoryForBank", "001"]}' -C myc
func (s *SmartContract) getHistoryForBank(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	BankID := args[0]

	fmt.Printf("- start getHistoryForBank: %s\n", BankID)

	resultsIterator, err := APIstub.GetHistoryForKey(BankID)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing historic values for the marble
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")
		// if it was a delete operation on given key, then we need to set the
		//corresponding value null. Else, we will write the response.Value
		//as-is (as the Value itself a JSON marble)
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getHistoryForBank returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc -c '{"Args":["getHistoryTXIDForBank","BANK002","a4723f60d5c85d29a2107382fb8e3c8c1624924b970efa04f313727a0dfaa0ff"]}' -C myc
func (s *SmartContract) getHistoryTXIDForBank(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	BankID := args[0]
	TXID := args[1]

	fmt.Printf("- start getHistoryTXIDForBank: %s\n", BankID)
	fmt.Printf("- start getHistoryTXIDForBank: %s\n", TXID)

	resultsIterator, err := APIstub.GetHistoryForKey(BankID)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing historic values for the marble
	var buffer bytes.Buffer
	buffer.WriteString("[")

	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		if response.TxId == TXID {
			buffer.WriteString("{\"TxId\":")
			buffer.WriteString("\"")
			buffer.WriteString(response.TxId)
			buffer.WriteString("\"")
			buffer.WriteString(", \"Value\":")
			// if it was a delete operation on given key, then we need to set the
			//corresponding value null. Else, we will write the response.Value
			//as-is (as the Value itself a JSON marble)
			if response.IsDelete {
				buffer.WriteString("null")
			} else {
				buffer.WriteString(string(response.Value))
			}

			buffer.WriteString(", \"Timestamp\":")
			buffer.WriteString("\"")
			buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
			buffer.WriteString("\"")

			buffer.WriteString(", \"IsDelete\":")
			buffer.WriteString("\"")
			buffer.WriteString(strconv.FormatBool(response.IsDelete))
			buffer.WriteString("\"")
			buffer.WriteString("}")

			break
		}
	}
	buffer.WriteString("]")

	fmt.Printf("- getHistoryTXIDForBank returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

func (s *SmartContract) queryAllBankKeys(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 2 {
		return shim.Error("Keys operation must include two arguments, startKey and endKey")
	}
	startKey := args[0]
	endKey := args[1]

	//sleep needed to test peer's timeout behavior when using iterators
	stime := 0
	if len(args) > 2 {
		stime, _ = strconv.Atoi(args[2])
	}

	keysIter, err := APIstub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
	}
	defer keysIter.Close()

	var keys []string
	for keysIter.HasNext() {
		//if sleeptime is specied, take a nap
		if stime > 0 {
			time.Sleep(time.Duration(stime) * time.Millisecond)
		}

		response, iterErr := keysIter.Next()
		if iterErr != nil {
			return shim.Error(fmt.Sprintf("keys operation failed. Error accessing state: %s", err))
		}
		keys = append(keys, response.Key)
	}

	for key, value := range keys {
		fmt.Printf("key %d contains %s\n", key, value)
	}

	jsonKeys, err := json.Marshal(keys)
	if err != nil {
		return shim.Error(fmt.Sprintf("keys operation failed. Error marshaling JSON: %s", err))
	}

	return shim.Success(jsonKeys)

}

func getBankStructFromID(
	stub shim.ChaincodeStubInterface,
	BankID string) (*Bank, error) {

	var errMsg string
	bank := &Bank{}
	bankAsBytes, err := stub.GetState(BankID)
	if err != nil {
		return bank, err
	} else if bankAsBytes == nil {
		errMsg = fmt.Sprintf("Error: BankID does not exist (%s)", BankID)
		return bank, errors.New(errMsg)
	}
	err = json.Unmarshal([]byte(bankAsBytes), bank)
	if err != nil {
		return bank, err
	}
	return bank, nil
}

func updateBankTotals(stub shim.ChaincodeStubInterface, BankID string, SecurityID string, AccountID string, Balance int64, Amount int64, isNegative bool) error {
	fmt.Printf("updateBankTotals: BankID=%s,SecurityID=%s,AccountID=%s,Balance=%d,Amount=%d\n", BankID, SecurityID, AccountID, Balance, Amount)
	TimeNow2 := time.Now().Format(timelayout2)
	newbankid := "BANK" + SubString(BankID, 0, 3)
	bank, err := getBankStructFromID(stub, newbankid)
	fmt.Printf("new BankID=%s\n", newbankid)
	if err != nil {
		return err
	}
	var doflg bool
	doflg = false
	var bankTotal BankTotal
	fmt.Printf("SecurityID=%s\n", SecurityID)
	for key, val := range bank.BankTotals {
		fmt.Printf("1.Bkey: %d\n", key)
		fmt.Printf("2.Bval: %s\n", val)
		if val.SecurityID == SecurityID {
			fmt.Printf("3.Bkey: %d\n", key)
			fmt.Printf("4.Bval: %s\n", val)
			if isNegative != true {
				bank.BankTotals[key].TotalBalance += Balance
				bank.BankTotals[key].TotalAmount += Amount
			} else if isNegative == true {
				bank.BankTotals[key].TotalBalance -= Balance
				bank.BankTotals[key].TotalAmount -= Amount
			}

			bank.BankTotals[key].UpdateTime = TimeNow2
			doflg = true
			break
		}
	}
	if doflg != true {
		bankTotal.SecurityID = SecurityID
		bankTotal.TotalBalance = Balance
		bankTotal.TotalAmount = Amount
		bankTotal.CreateTime = TimeNow2
		bankTotal.UpdateTime = TimeNow2
		bank.BankTotals = append(bank.BankTotals, bankTotal)
	}
	doflg = false
	for key, val := range bank.BankAccounts {
		fmt.Printf("akey1: %d\n", key)
		fmt.Printf("aval1: %s\n", val)

		if val == AccountID {
			fmt.Printf("akey2: %d\n", key)
			fmt.Printf("aval2: %s\n", val)
			doflg = true
			break
		}
	}
	if doflg != true {
		bank.BankAccounts = append(bank.BankAccounts, AccountID)
	}

	bankAsBytes, err := json.Marshal(bank)
	if err != nil {
		return err
	}
	err = stub.PutState(newbankid, bankAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateBankAccounts(stub shim.ChaincodeStubInterface, BankID string, AccountID string) error {
	fmt.Printf("updateBankAccounts: BankID=%s,AccountID=%s\n", BankID, AccountID)

	newbankid := "BANK" + SubString(BankID, 0, 3)
	bank, err := getBankStructFromID(stub, newbankid)
	fmt.Printf("new BankID=%s\n", newbankid)
	if err != nil {
		return err
	}

	var doflg bool
	doflg = false

	for key, val := range bank.BankAccounts {
		fmt.Printf("akey1: %d\n", key)
		fmt.Printf("aval1: %s\n", val)

		if val == AccountID {
			fmt.Printf("akey2: %d\n", key)
			fmt.Printf("aval2: %s\n", val)
			doflg = true
			break
		}
	}
	if doflg != true {
		bank.BankAccounts = append(bank.BankAccounts, AccountID)
	}

	bankAsBytes, err := json.Marshal(bank)
	if err != nil {
		return err
	}
	err = stub.PutState(newbankid, bankAsBytes)
	if err != nil {
		return err
	}
	return nil
}

//peer chaincode query -n mycc -c '{"Args":["BankTotals","BANK002"]}' -C myc
func (s *SmartContract) queryBankTotals(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	BankAsBytes, _ := APIstub.GetState(args[0])
	Bank := Bank{}
	json.Unmarshal(BankAsBytes, &Bank)

	BankTotalsAsBytes, err := json.Marshal(Bank.BankTotals)
	if err != nil {
		return shim.Error("Failed to query BankTotals state")
	}

	return shim.Success(BankTotalsAsBytes)
}

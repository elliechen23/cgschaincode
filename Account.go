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

const accountObjectType string = "account"

type Asset struct {
	SecurityID     string `json:"SecurityID"`
	SecurityAmount int64  `json:"SecurityAmount"`
	Balance        int64  `json:"Balance"`
	Position       int64  `json:"Position"`
}

type Account struct {
	ObjectType string  `json:"docType"`   // default set to "account"
	AccountID  string  `json:"AccountID"` // account ID
	BankID     string  `json:"BankID"`    // the fieldtags are needed to keep case from bouncing around
	BankName   string  `json:"BankName"`
	CustName   string  `json:CustName` //客戶名稱
	CustType   string  `json:CustType` //存戶類別編號
	Status     string  `json:"Status"` // Status values ( NORMAL, PAUSED )
	Assets     []Asset `json:"Assets"`
}

//peer chaincode invoke -n mycc2 -c '{"Args":["initAccount", "002000000001" , "002" , "BANK002" , "CUST001" , "00001", "A06101", "1000000" , "1000000" , "1000000", "0" ]}' -C myc
func (s *SmartContract) initAccount(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	// AccountID, BankID,SecurityID, Balance, Status
	err := checkArgArrayLength(args, 10)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("AccountID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("BankID must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("BankName must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return shim.Error("CustName must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return shim.Error("CustType must be a non-empty string")
	}
	if len(args[5]) <= 0 {
		return shim.Error("SecurityID must be a non-empty string")
	}
	if len(args[6]) <= 0 {
		return shim.Error("SecurityAmount must be a non-empty string")
	}
	if len(args[7]) <= 0 {
		return shim.Error("Balance must be a non-empty string")
	}
	if len(args[8]) <= 0 {
		return shim.Error("Position must be a non-empty string")
	}
	if len(args[9]) <= 0 {
		return shim.Error("Status must be a non-empty string")
	}

	AccountID := args[0]
	BankID := strings.ToUpper(args[1])
	BankName := strings.ToUpper(args[2])
	CustName := strings.ToUpper(args[3])
	CustType := strings.ToUpper(args[4])
	SecurityID := strings.ToUpper(args[5])
	SecurityAmount, err := strconv.ParseInt(args[6], 10, 64)
	if err != nil {
		return shim.Error("SecurityAmount must be a numeric string")
	}
	Balance, err := strconv.ParseInt(args[7], 10, 64)
	if err != nil {
		return shim.Error("Balance must be a numeric string")
	}
	Position, err := strconv.ParseInt(args[8], 10, 64)
	if err != nil {
		return shim.Error("Position must be a numeric string")
	}
	Status := strings.ToUpper(args[9])
	accountAsBytes, err := stub.GetState(AccountID)
	if err != nil {
		return shim.Error(err.Error())
	} else if accountAsBytes != nil {
		errMsg := fmt.Sprintf(
			"Error: This account already exists (%s)",
			AccountID)
		return shim.Error(errMsg)
	}
	var asset Asset
	asset.SecurityID = SecurityID
	asset.SecurityAmount = SecurityAmount
	asset.Balance = Balance
	asset.Position = Position

	account := Account{}
	account.ObjectType = accountObjectType
	account.AccountID = AccountID
	account.BankID = BankID
	account.BankName = BankName
	account.CustName = CustName
	account.CustType = CustType
	account.Status = Status
	account.Assets = append(account.Assets, asset)

	accountAsBytes, err = json.Marshal(account)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(AccountID, accountAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(accountAsBytes)
}

//peer chaincode invoke -n mycc2 -c '{"Args":["updateAccount", "002000000001" , "002" , "BANK002" , "CUST001" , "00001", "A06101", "1010000" , "1000000" , "1000000", "0" ]}' -C myc
//peer chaincode invoke -n mycc2 -c '{"Args":["updateAccount", "002000000001" , "002" , "BANK002" , "CUST001" , "00001", "A06102", "1020000" , "1000000" , "1000000", "0" ]}' -C myc

func (s *SmartContract) updateAccount(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	// AccountID, BankID, Balance, Status
	err := checkArgArrayLength(args, 10)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("AccountID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("BankID must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("BankName must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return shim.Error("CustName must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return shim.Error("CustType must be a non-empty string")
	}
	if len(args[5]) <= 0 {
		return shim.Error("SecurityID must be a non-empty string")
	}
	if len(args[6]) <= 0 {
		return shim.Error("SecurityAmount must be a non-empty string")
	}
	if len(args[7]) <= 0 {
		return shim.Error("Balance must be a non-empty string")
	}
	if len(args[8]) <= 0 {
		return shim.Error("Position must be a non-empty string")
	}
	if len(args[9]) <= 0 {
		return shim.Error("Status must be a non-empty string")
	}

	AccountID := args[0]
	BankID := strings.ToUpper(args[1])
	BankName := strings.ToUpper(args[2])
	CustName := strings.ToUpper(args[3])
	CustType := strings.ToUpper(args[4])
	SecurityID := strings.ToUpper(args[5])
	SecurityAmount, err := strconv.ParseInt(args[6], 10, 64)
	if err != nil {
		return shim.Error("SecurityAmount must be a numeric string")
	}
	Balance, err := strconv.ParseInt(args[7], 10, 64)
	if err != nil {
		return shim.Error("Balance must be a numeric string")
	}
	Position, err := strconv.ParseInt(args[8], 10, 64)
	if err != nil {
		return shim.Error("Position must be a numeric string")
	}
	Status := strings.ToUpper(args[9])
	accountAsBytes, err := stub.GetState(AccountID)
	if err != nil {
		return shim.Error(err.Error())
	} else if accountAsBytes == nil {
		errMsg := fmt.Sprintf(
			"Error: This account does not exist (%s)",
			AccountID)
		return shim.Error(errMsg)
	}

	account := Account{}
	json.Unmarshal(accountAsBytes, &account)
	account.ObjectType = accountObjectType
	account.AccountID = AccountID
	account.BankID = BankID
	account.BankName = BankName
	account.CustName = CustName
	account.CustType = CustType
	account.Status = Status

	var doflg bool
	doflg = false
	for key, val := range account.Assets {
		if val.SecurityID == SecurityID {
			account.Assets[key].SecurityID = SecurityID
			account.Assets[key].SecurityAmount = SecurityAmount
			account.Assets[key].Balance = Balance
			account.Assets[key].Position = Position
			doflg = true
			break
		}
	}
	if doflg != true {
		var asset Asset
		asset.SecurityID = SecurityID
		asset.SecurityAmount = SecurityAmount
		asset.Balance = Balance
		asset.Position = Position
		account.Assets = append(account.Assets, asset)
	}

	accountAsBytes, err = json.Marshal(account)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = stub.PutState(AccountID, accountAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(accountAsBytes)
}

func (s *SmartContract) deleteAccount(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	err := checkArgArrayLength(args, 2)
	if err != nil {
		return shim.Error(err.Error())
	}

	AccountID := args[0]
	BankID := args[1]

	// Access Control
	//err = verifyIdentity(stub, BankID)
	//if err != nil {
	//	return shim.Error(err.Error())
	//}

	accountAsBytes, err := stub.GetState(AccountID)
	if err != nil {
		errMsg := fmt.Sprintf(
			"Error: Failed to get state for account (%s)",
			AccountID)
		return shim.Error(errMsg)
	} else if accountAsBytes == nil {
		errMsg := fmt.Sprintf(
			"Error: Account does not exist (%s)",
			AccountID)
		return shim.Error(errMsg)
	}

	account := Account{}
	json.Unmarshal(accountAsBytes, &account)

	if account.BankID != BankID {
		errMsg := fmt.Sprintf(
			"bankID set for account [%s] does not match BankID provided [%s]",
			account.BankID,
			BankID)
		return shim.Error(errMsg)
	}

	err = stub.DelState(AccountID)
	if err != nil {
		return shim.Error("Failed to delete state:" + err.Error())
	}
	return shim.Success(nil)
}

//Status:PAUSED, NORMAL
//peer chaincode invoke -n mycc2 -c '{"Args":["updateAccountStatus", "002000000001" , "NORMAL"]}' -C myc
func (s *SmartContract) updateAccountStatus(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	err := checkArgArrayLength(args, 2)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("AccountID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("Status must be a non-empty string")
	}

	AccountID := args[0]
	account, err := getAccountStructFromID(stub, AccountID)
	account.Status = args[1]

	accountAsBytes, err := json.Marshal(account)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(AccountID, accountAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

//peer chaincode query -n mycc2 -c '{"Args":["readAccount","002000000001"]}' -C myc
func (s *SmartContract) getStateAsBytes(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	err := checkArgArrayLength(args, 1)
	if err != nil {
		return shim.Error(err.Error())
	}

	key := args[0]
	valAsbytes, err := stub.GetState(key)
	if err != nil {
		return shim.Error(err.Error())
	} else if valAsbytes == nil {
		errMsg := fmt.Sprintf("Error: Key does not exist (%s)", key)
		return shim.Error(errMsg)
	}

	return shim.Success(valAsbytes)
}

func getAccountStructFromID(
	stub shim.ChaincodeStubInterface,
	AccountID string) (*Account, error) {

	var errMsg string
	account := &Account{}
	accountAsBytes, err := stub.GetState(AccountID)
	if err != nil {
		return account, err
	} else if accountAsBytes == nil {
		errMsg = fmt.Sprintf("Error: Account does not exist (%s)", AccountID)
		return account, errors.New(errMsg)
	}
	err = json.Unmarshal([]byte(accountAsBytes), account)
	if err != nil {
		return account, err
	}
	return account, nil
}

func checkArgArrayLength(
	args []string,
	expectedArgLength int) error {

	argArrayLength := len(args)
	if argArrayLength != expectedArgLength {
		errMsg := fmt.Sprintf(
			"Incorrect number of arguments: Received %d, expecting %d",
			argArrayLength,
			expectedArgLength)
		return errors.New(errMsg)
	}
	return nil
}

/*
peer chaincode invoke -n mycc2 -c '{"Args":["updateAccount", "002000000001" , "002" , "BANK002" , "CUST001" , "00001", "A06101", "1010000" , "1000000" , "1000000", "0" ]}' -C myc
peer chaincode invoke -n mycc2 -c '{"Args":["updateAccount", "002000000001" , "002" , "BANK002" , "CUST001" , "00001", "A06102", "1020000" , "1000000" , "1000000", "0" ]}' -C myc
*/
func (s *SmartContract) updateAsset(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	err := checkArgArrayLength(args, 5)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("AccountID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("SecurityID must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("SecurityAmount must be a non-empty string")
	}
	if len(args[3]) <= 0 {
		return shim.Error("Balance must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return shim.Error("Position must be a non-empty string")
	}

	AccountID := args[0]
	SecurityID := args[1]
	SecurityAmount, err := strconv.ParseInt(args[2], 10, 64)
	if err != nil {
		return shim.Error("SecurityAmount must be a numeric string")
	}
	Balance, err := strconv.ParseInt(args[3], 10, 64)
	if err != nil {
		return shim.Error("Balance must be a numeric string")
	}
	Position, err := strconv.ParseInt(args[4], 10, 64)
	if err != nil {
		return shim.Error("Position must be a numeric string")
	}

	account, err := getAccountStructFromID(stub, AccountID)
	if account.Status == "PAUSED" {
		return shim.Error("Account Status is : " + account.Status)
	}

	var doflg bool
	doflg = false
	for key, val := range account.Assets {
		if val.SecurityID == args[1] {
			account.Assets[key].SecurityAmount = SecurityAmount
			account.Assets[key].Balance = Balance
			account.Assets[key].Position = Position
			doflg = true
		}
	}

	if doflg != true {
		var asset Asset
		asset.SecurityID = SecurityID
		asset.SecurityAmount = SecurityAmount
		asset.Balance = Balance
		asset.Position = Position
	}
	accountAsBytes, err := json.Marshal(account)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(AccountID, accountAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (s *SmartContract) updateAssetBalance(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	err := checkArgArrayLength(args, 5)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("AccountID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("SecurityID must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return shim.Error("BUY or SELL must be a non-empty string") //BUY , SELL
	}
	if len(args[3]) <= 0 {
		return shim.Error("Balance must be a non-empty string")
	}
	if len(args[4]) <= 0 {
		return shim.Error("Position must be a non-empty string")
	}

	AccountID := strings.ToUpper(args[0])
	SecurityID := strings.ToUpper(args[1])
	BuyOrSell := strings.ToUpper(args[2])
	Balance, err := strconv.ParseInt(args[3], 10, 64)
	if err != nil {
		return shim.Error("Balance must be a numeric string")
	}
	Position, err := strconv.ParseInt(args[4], 10, 64)
	if err != nil {
		return shim.Error("Position must be a numeric string")
	}

	account, err := getAccountStructFromID(stub, AccountID)
	if account.Status == "PAUSED" {
		return shim.Error("Account Status is : " + account.Status)
	}

	var doflg bool
	doflg = false
	for key, val := range account.Assets {
		if val.SecurityID == SecurityID {
			if BuyOrSell == "SELL" {
				account.Assets[key].Balance -= Balance
				account.Assets[key].Position -= Position
			}
			if BuyOrSell == "BUY" {
				account.Assets[key].Balance += Balance
				account.Assets[key].Position += Position
			}
			doflg = true
		}
	}

	if doflg != true {
		return shim.Error("Failed to query assets state")
	}
	accountAsBytes, err := json.Marshal(account)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(AccountID, accountAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

//peer chaincode invoke -n mycc -c '{"Args":["deleteAsset", "002000000001" , "A06101" ]}' -C myc
func (s *SmartContract) deleteAsset(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	err := checkArgArrayLength(args, 2)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("AccountID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("SecurityID must be a non-empty string")
	}

	AccountID := args[0]
	account, err := getAccountStructFromID(stub, AccountID)

	var doflg bool
	doflg = false
	var assets []Asset
	filteredAssets := assets[:0]
	for _, val := range account.Assets {
		if val.SecurityID == args[1] {
			doflg = true
		} else {
			filteredAssets = append(filteredAssets, val)
		}
	}

	if doflg == true {
		account.Assets = filteredAssets
	}
	accountAsBytes, err := json.Marshal(account)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(AccountID, accountAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

//peer chaincode query -n mycc2 -c '{"Args":["queryAsset","002000000001"]}' -C myc
func (s *SmartContract) queryAsset(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	AccountAsBytes, _ := APIstub.GetState(args[0])
	Account := Account{}
	json.Unmarshal(AccountAsBytes, &Account)

	OwnerAsBytes, err := json.Marshal(Account.Assets)
	if err != nil {
		return shim.Error("Failed to query assets state")
	}

	return shim.Success(OwnerAsBytes)
}

//peer chaincode query -n mycc2 -c '{"Args":["queryAssetLength","002000000001"]}' -C myc
func (s *SmartContract) queryAssetLength(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	AccountAsBytes, _ := APIstub.GetState(args[0])
	Account := Account{}
	json.Unmarshal(AccountAsBytes, &Account)

	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"AccountID\":")
	buffer.WriteString("\"")
	buffer.WriteString(Account.AccountID)
	buffer.WriteString("\"")
	buffer.WriteString(", \"OwnersLength\":")
	buffer.WriteString(strconv.Itoa(len(Account.Assets)))
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())

}

//peer chaincode query -n mycc2 -c '{"Args":["queryAssetInfo","002000000001" , "A06101" ]}' -C myc
func (s *SmartContract) queryAssetInfo(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	AccountAsBytes, _ := APIstub.GetState(args[0])
	Account := Account{}
	json.Unmarshal(AccountAsBytes, &Account)

	var doflg bool
	doflg = false
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"AccountID\":")
	buffer.WriteString("\"")
	buffer.WriteString(Account.AccountID)
	buffer.WriteString("\"")

	for key, val := range Account.Assets {
		if val.SecurityID == args[1] {
			buffer.WriteString(", \"AssetKey\":")
			buffer.WriteString(strconv.Itoa(key))
			buffer.WriteString(", \"SecurityID\":")
			buffer.WriteString(val.SecurityID)
			buffer.WriteString(", \"SecurityAmount\":")
			buffer.WriteString(strconv.FormatInt(val.SecurityAmount, 10))
			buffer.WriteString(", \"Balance\":")
			buffer.WriteString(strconv.FormatInt(val.Balance, 10))
			buffer.WriteString(", \"Position\":")
			buffer.WriteString(strconv.FormatInt(val.Position, 10))
			doflg = true
		}
	}
	if doflg != true {
		return shim.Error("Failed to find SecurityID ")
	}
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc2 -c '{"Args":["queryAccountStatus","002000000001"]}' -C myc
func (s *SmartContract) queryAccountStatus(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	AccountAsBytes, _ := APIstub.GetState(args[0])
	Account := Account{}
	json.Unmarshal(AccountAsBytes, &Account)

	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"AccountID\":")
	buffer.WriteString("\"")
	buffer.WriteString(Account.AccountID)
	buffer.WriteString("\"")
	buffer.WriteString(", \"AccountStatus\":")
	buffer.WriteString(Account.Status)
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())

}

//peer chaincode query -n mycc2 -c '{"Args":["queryAllAccounts","000000000001" , "999999999999"]}' -C myc
func (s *SmartContract) queryAllAccounts(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

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

//peer chaincode query -n mycc -c '{"Args":["getHistoryForAccount","002000000001"]}' -C myc
func (s *SmartContract) getHistoryForAccount(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	AccountID := args[0]

	fmt.Printf("- start getHistoryForAccount: %s\n", AccountID)

	resultsIterator, err := APIstub.GetHistoryForKey(AccountID)

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

	fmt.Printf("- getHistoryForAccount returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc -c '{"Args":["queryAllAccountKeys","002000000001" , "004000000009"]}' -C myc
//Query Result: ["002000000001","002000000002","004000000001","004000000002"]

func (s *SmartContract) queryAllAccountKeys(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

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

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright SecurityNameship.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE/2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

/*
 * The sample smart contract for documentation topic:
 * Writing Your First Blockchain Application
 */
/*
Building Chaincode
Now let’s compile your chaincode.

go get -u --tags nopkcs11 github.com/hyperledger/fabric/core/chaincode/shim
go build --tags nopkcs11
*/

package main

/* Imports
 * 4 utility libraries for formatting, handling bytes, reading and writing JSON, and string manipulation
 * 2 specific Hyperledger Fabric specific libraries for Smart Contracts
 */
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

// Define the Smart Contract structure
type SmartContract struct {
}

const (
	millisPerSecond       = int64(time.Second / time.Millisecond)
	nanosPerMillisecond   = int64(time.Millisecond / time.Nanosecond)
	layout                = "2006/01/02"
	unitAmount            = int64(1000000)       //1單位=100萬
	perDayMillionInterest = float64(27.397)      //每1百萬面額，利率=1%，一天的利息
	perDayInterest        = float64(0.000027397) //每1元面額，利率=1%，一天的利息
	//InterestObjectType    = "Interest"
)

type SecurityTotal struct {
	BankID               string `json:"BankID"`
	TotalBalance         int64  `json:"TotalBalance"`
	TotalInterest        int64  `json:"TotalInterest"`
	DurationInterest     int64  `json:"DurationInterest"`
	PaidDurationInterest int64  `json:"PaidDurationInterest"`
	CreateTime           string `json:"CreateTime"`
	UpdateTime           string `json:"UpdateTime"`
}

type Owner struct {
	OwnedAccountID        string   `json:"OwnedAccountID"`
	OwnedBankID           string   `json:"OwnedBankID"`
	OwnedAmount           int64    `json:"OwnedAmount"`
	OwnedRepay            int64    `json:"OwnedRepay"`
	OwnedInterest         int64    `json:"OwnedInterest"`
	OwnedDurationInterest int64    `json:"OwnedDurationInterest"`
	OwnedDurationDate     []string `json:"OwnedDurationDate"`
	Avaliable             int      `json:"Avaliable"`
}

/*
1.登錄之清算銀行帳號：000(三碼)+流水號(九碼)
2.登錄之清算銀行代號：000(三碼)
3.登錄之公債面額：______(100萬)
4.登錄之公債還本付息：______(=公債面額+利息)
5.登錄之公債利息：______(天數*每天利息*票面利率*單位數(per 100萬))
6.登錄之公債利息(每一期)：______(登錄之公債利息/公債期數)
7.登錄之公債期數日期：________
8.登錄之可用性？
*/

//Book/Entry Central Government Securities (CGS)中央登錄公債
// Define the Security structure, with 7 properties.  Structure tags are used by encoding/json library
type Security struct {
	ObjectType           string          `json:"docType"` //docType is used to distinguish the various types of objects in state database
	SecurityID           string          `json:"SecurityID"`
	SecurityName         string          `json:"SecurityName"`
	IssueDate            string          `json:"IssueDate"`
	MaturityDate         string          `json:"MaturityDate"`
	InterestRate         float64         `json:"InterestRate"`
	RepayPeriod          int             `json:"RepayPeriod"`
	TotalAmount          int64           `json:"TotalAmount"`
	Balance              int64           `json:"Balance"`
	SecurityStatus       int             `json:"SecurityStatus"`
	Owners               []Owner         `json:"Owners"`
	SecurityTotals       []SecurityTotal `json:"SecurityTotals"`
	SecurityDurationDate []string        `json:"SecurityDurationDate"`
}

/*
 1.公債代號︰______
 2.公債名稱︰___________________________________
 3.發 行 日︰_______
 4.到 期 日︰_______
 5.票面利率︰__.______
 6.公債年期：__ 年
 7.公債發行總額：_______(250億)
 8.公債剩餘總額：_______
 9.公債狀態：__
 10.登錄之清算銀行清單：
 11.公債持有銀行的總額：
 11.公債每一期付息的日期：
*/

/*
 * The Init method is called when the Smart Contract "CGSecurity" is instantiated by the blockchain network
 * Best practice is to have any Ledger initialization in sepaRate function // see initLedger()
 */
func (s *SmartContract) Init(APIstub shim.ChaincodeStubInterface) peer.Response {

	return shim.Success(nil)
}

/*
 * The Invoke method is called as a result of an application request to run the Smart Contract "CGSecurity"
 * The calling application program has also specified the particular smart contract function to be called, with arguments
 */
func (s *SmartContract) Invoke(APIstub shim.ChaincodeStubInterface) peer.Response {

	// Retrieve the requested Smart Contract function and arguments
	function, args := APIstub.GetFunctionAndParameters()
	// Route to the appropriate handler function to interact with the ledger appropriately
	if function == "querySecurity" {
		return s.querySecurity(APIstub, args)
	} else if function == "initLedger" {
		return s.initLedger(APIstub, args)
	} else if function == "createSecurity" {
		return s.createSecurity(APIstub, args)
	} else if function == "queryAllSecurities" {
		return s.queryAllSecurities(APIstub, args)
	} else if function == "querySecurityStatus" {
		return s.querySecurityStatus(APIstub, args)
	} else if function == "queryOwner" {
		return s.queryOwner(APIstub, args)
	} else if function == "queryOwnerAccount" {
		return s.queryOwnerAccount(APIstub, args)
	} else if function == "queryOwnerLength" {
		return s.queryOwnerLength(APIstub, args)
	} else if function == "changeSecurity" {
		return s.changeSecurity(APIstub, args)
	} else if function == "changeSecurityStatus" {
		return s.changeSecurityStatus(APIstub, args)
	} else if function == "changeOwnerAvaliable" {
		return s.changeOwnerAvaliable(APIstub, args)
	} else if function == "deleteSecurity" {
		return s.deleteSecurity(APIstub, args)
	} else if function == "deleteOwner" {
		return s.deleteOwner(APIstub, args)
	} else if function == "updateOwnerInterest" {
		return s.updateOwnerInterest(APIstub, args)
	} else if function == "getHistoryForSecurity" {
		return s.getHistoryForSecurity(APIstub, args)
	} else if function == "queryAllSecurityKeys" {
		return s.queryAllSecurityKeys(APIstub, args)
	} else if function == "updateSecurityTotals" {
		return s.updateSecurityTotals(APIstub, args)
		// Account Functions
	} else if function == "initAccount" {
		return s.initAccount(APIstub, args)
	} else if function == "deleteAccount" {
		return s.deleteAccount(APIstub, args)
	} else if function == "readAccount" {
		return s.getStateAsBytes(APIstub, args)
	} else if function == "updateAccountStatus" {
		return s.updateAccountStatus(APIstub, args)
	} else if function == "updateAccount" {
		return s.updateAccount(APIstub, args)
	} else if function == "updateAsset" {
		return s.updateAsset(APIstub, args)
	} else if function == "updateAssetBalance" {
		return s.updateAssetBalance(APIstub, args)
	} else if function == "deleteAsset" {
		return s.deleteAsset(APIstub, args)
	} else if function == "queryAsset" {
		return s.queryAsset(APIstub, args)
	} else if function == "queryAssetInfo" {
		return s.queryAssetInfo(APIstub, args)
	} else if function == "queryAssetLength" {
		return s.queryAssetLength(APIstub, args)
	} else if function == "queryAccountStatus" {
		return s.queryAccountStatus(APIstub, args)
	} else if function == "queryAllAccounts" {
		return s.queryAllAccounts(APIstub, args)
	} else if function == "getHistoryForAccount" {
		return s.getHistoryForAccount(APIstub, args)
	} else if function == "queryAllAccountKeys" {
		return s.queryAllAccountKeys(APIstub, args)
		// Bank Functions
	} else if function == "initBank" {
		return s.initBank(APIstub, args)
	} else if function == "updateBank" {
		return s.updateBank(APIstub, args)
	} else if function == "deleteBank" {
		return s.deleteBank(APIstub, args)
	} else if function == "verifyBankList" {
		return s.verifyBankList(APIstub, args)
	} else if function == "readBank" {
		return s.getStateAsBytes(APIstub, args)
	} else if function == "queryAllBanks" {
		return s.queryAllBanks(APIstub, args)
	} else if function == "getHistoryForBank" {
		return s.getHistoryForBank(APIstub, args)
	} else if function == "queryAllBankKeys" {
		return s.queryAllBankKeys(APIstub, args)
		// Transaction Functions
	} else if function == "submitApproveTransaction" {
		return s.submitApproveTransaction(APIstub, args)
	} else if function == "submitEndDayTransaction" {
		return s.submitEndDayTransaction(APIstub, args)
	} else if function == "securityTransfer" {
		return s.securityTransfer(APIstub, args)
	} else if function == "securityCorrectTransfer" {
		return s.securityCorrectTransfer(APIstub, args)
	} else if function == "queryTXIDTransactions" {
		return s.queryTXIDTransactions(APIstub, args)
	} else if function == "queryTXKEYTransactions" {
		return s.queryTXKEYTransactions(APIstub, args)
	} else if function == "queryHistoryTXKEYTransactions" {
		return s.queryHistoryTXKEYTransactions(APIstub, args)
	} else if function == "getHistoryForTransaction" {
		return s.getHistoryForTransaction(APIstub, args)
	} else if function == "getHistoryForQueuedTransaction" {
		return s.getHistoryForQueuedTransaction(APIstub, args)
	} else if function == "queryAllTransactions" {
		return s.queryAllTransactions(APIstub, args)
	} else if function == "queryAllQueuedTransactions" {
		return s.queryAllQueuedTransactions(APIstub, args)
	} else if function == "queryAllHistoryTransactions" {
		return s.queryAllHistoryTransactions(APIstub, args)
	} else if function == "queryAllTransactionKeys" {
		return s.queryAllTransactionKeys(APIstub, args)
	} else {
		//map functions
		return s.mapFunction(APIstub, function, args)
	}

	return shim.Error("Invalid Smart Contract function name.")
}

func (s *SmartContract) mapFunction(stub shim.ChaincodeStubInterface, function string, args []string) peer.Response {
	switch function {

	case "put":
		if len(args) < 2 {
			return shim.Error("put operation must include two arguments: [key, value]")
		}
		key := args[0]
		value := args[1]

		if err := stub.PutState(key, []byte(value)); err != nil {
			fmt.Printf("Error putting state %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state: %s", err))
		}

		indexName := "compositeKeyTest"
		compositeKeyTestIndex, err := stub.CreateCompositeKey(indexName, []string{key})
		if err != nil {
			return shim.Error(err.Error())
		}

		valueByte := []byte{0x00}
		if err := stub.PutState(compositeKeyTestIndex, valueByte); err != nil {
			fmt.Printf("Error putting state with compositeKey %s", err)
			return shim.Error(fmt.Sprintf("put operation failed. Error updating state with compositeKey: %s", err))
		}

		return shim.Success(nil)

	case "remove":
		if len(args) < 1 {
			return shim.Error("remove operation must include one argument: [key]")
		}
		key := args[0]

		err := stub.DelState(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("remove operation failed. Error updating state: %s", err))
		}
		return shim.Success(nil)

	case "get":
		if len(args) < 1 {
			return shim.Error("get operation must include one argument, a key")
		}
		key := args[0]
		value, err := stub.GetState(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("get operation failed. Error accessing state: %s", err))
		}
		jsonVal, err := json.Marshal(string(value))
		return shim.Success(jsonVal)

	case "keys":
		if len(args) < 2 {
			return shim.Error("put operation must include two arguments, a key and value")
		}
		startKey := args[0]
		endKey := args[1]

		//sleep needed to test peer's timeout behavior when using iterators
		stime := 0
		if len(args) > 2 {
			stime, _ = strconv.Atoi(args[2])
		}

		keysIter, err := stub.GetStateByRange(startKey, endKey)
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
	case "query":
		query := args[0]
		keysIter, err := stub.GetQueryResult(query)
		if err != nil {
			return shim.Error(fmt.Sprintf("query operation failed. Error accessing state: %s", err))
		}
		defer keysIter.Close()

		var keys []string
		for keysIter.HasNext() {
			response, iterErr := keysIter.Next()
			if iterErr != nil {
				return shim.Error(fmt.Sprintf("query operation failed. Error accessing state: %s", err))
			}
			keys = append(keys, response.Key)
		}

		jsonKeys, err := json.Marshal(keys)
		if err != nil {
			return shim.Error(fmt.Sprintf("query operation failed. Error marshaling JSON: %s", err))
		}

		return shim.Success(jsonKeys)
	case "history":
		key := args[0]
		keysIter, err := stub.GetHistoryForKey(key)
		if err != nil {
			return shim.Error(fmt.Sprintf("query operation failed. Error accessing state: %s", err))
		}
		defer keysIter.Close()

		var keys []string
		for keysIter.HasNext() {
			response, iterErr := keysIter.Next()
			if iterErr != nil {
				return shim.Error(fmt.Sprintf("query operation failed. Error accessing state: %s", err))
			}
			keys = append(keys, response.TxId)
		}

		for key, txID := range keys {
			fmt.Printf("key %d contains %s\n", key, txID)
		}

		jsonKeys, err := json.Marshal(keys)
		if err != nil {
			return shim.Error(fmt.Sprintf("query operation failed. Error marshaling JSON: %s", err))
		}

		return shim.Success(jsonKeys)

	default:
		return shim.Success([]byte("Unsupported operation"))
	}
}

/*
 107年02月18日 中央登錄公債資料表
 公債代號	公債簡稱	 發行日期	到期日	     票面利率    年期
 A06101    	106甲01  2017/01/11	2019/01/11	0.5      2
 A06102    	106甲02  2017/01/23	2022/01/23	0.75     5
 A06103    	106甲03  2017/02/20	2037/02/20	1.75     20
 A06104    	106甲04  2017/03/01	2027/03/01	1.125    10
 A06105    	106甲05  2017/04/21	2022/04/21	0.75     5
 A06106    	106甲06  2017/05/26	2047/05/26	1.875    30
 A06107    	106甲07  2017/07/27	2019/07/27	0.5      2
 A06108    	106甲08  2017/08/18	2037/08/18	1.5      20
 A06109    	106甲09  2017/09/20	2027/09/20	1        10
 A06110    	106甲10  2017/10/18  2022/10/18	0.625    5
 A06111    	106甲11  2017/11/24  2047/11/24	1.625    30
 A07101    	107甲01  2018/01/12	2023/01/12	0.625    5
 A07102    	107甲02  2018/02/08	2028/02/08	1        10
*/

/*
 peer chaincode invoke -n mycc -c '{"Args":["initLedger", "004"]}' -C myc
*/
func (s *SmartContract) initLedger(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	Securities := []Security{
		Security{ObjectType: "security", SecurityID: "A06101", SecurityName: "106甲01", IssueDate: "2017/01/11", MaturityDate: "2019/01/11", InterestRate: 0.5, RepayPeriod: 2, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06102", SecurityName: "106甲02", IssueDate: "2017/01/23", MaturityDate: "2022/01/23", InterestRate: 0.75, RepayPeriod: 5, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06103", SecurityName: "106甲03", IssueDate: "2017/02/20", MaturityDate: "2037/02/20", InterestRate: 1.75, RepayPeriod: 20, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06104", SecurityName: "106甲04", IssueDate: "2017/03/01", MaturityDate: "2027/03/01", InterestRate: 1.125, RepayPeriod: 10, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06105", SecurityName: "106甲05", IssueDate: "2017/04/21", MaturityDate: "2022/04/21", InterestRate: 0.75, RepayPeriod: 5, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06106", SecurityName: "106甲06", IssueDate: "2017/05/26", MaturityDate: "2019/05/26", InterestRate: 0.5, RepayPeriod: 2, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06107", SecurityName: "106甲07", IssueDate: "2017/07/27", MaturityDate: "2019/07/27", InterestRate: 0.5, RepayPeriod: 2, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06108", SecurityName: "106甲08", IssueDate: "2017/08/18", MaturityDate: "2037/08/18", InterestRate: 1.5, RepayPeriod: 20, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06109", SecurityName: "106甲09", IssueDate: "2017/09/20", MaturityDate: "2027/09/20", InterestRate: 1, RepayPeriod: 10, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06110", SecurityName: "106甲10", IssueDate: "2017/10/18", MaturityDate: "2022/10/18", InterestRate: 0.625, RepayPeriod: 5, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A06111", SecurityName: "106甲11", IssueDate: "2017/11/24", MaturityDate: "2047/11/24", InterestRate: 1.625, RepayPeriod: 30, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A07101", SecurityName: "107甲01", IssueDate: "2018/01/12", MaturityDate: "2023/01/12", InterestRate: 0.625, RepayPeriod: 5, SecurityStatus: 0},
		Security{ObjectType: "security", SecurityID: "A07102", SecurityName: "107甲02", IssueDate: "2018/02/08", MaturityDate: "2028/02/08", InterestRate: 1, RepayPeriod: 10, SecurityStatus: 0},
	}

	i := 0
	for i < len(Securities) {
		//fmt.Println("i is ", i)
		var owner Owner
		var securityTotal SecurityTotal

		TimeNow := time.Now().Format(timelayout)
		Today := SubString(TimeNow, 0, 8)

		if i < 9 {
			owner.OwnedAccountID = args[0] + SubString("00000000"+strconv.Itoa(i+1), 0, 9)
		} else if i >= 9 {
			owner.OwnedAccountID = args[0] + SubString("0000000"+strconv.Itoa(i+1), 0, 9)
		}

		owner.OwnedBankID = args[0]
		owner.OwnedAmount = unitAmount
		owner.OwnedInterest = int64(round(perDayMillionInterest*daySub(Securities[i].IssueDate, Securities[i].MaturityDate)*Securities[i].InterestRate, 0))
		owner.OwnedDurationInterest = owner.OwnedInterest / int64(Securities[i].RepayPeriod)
		j := 0
		var SecurityDurationDate []string
		var PaidDurationPeriod int64
		PaidDurationPeriod = 0

		for j < Securities[i].RepayPeriod {
			NextPayInterestDate, _ := generateMaturity(Securities[i].IssueDate, j+1, 0, 0)
			owner.OwnedDurationDate = append(owner.OwnedDurationDate, NextPayInterestDate)
			SecurityDurationDate = append(SecurityDurationDate, NextPayInterestDate)
			if Today >= NextPayInterestDate {
				PaidDurationPeriod = int64(j + 1)
			}
			j = j + 1
		}
		PaidDurationInterest := securityTotal.DurationInterest * PaidDurationPeriod
		owner.OwnedRepay = unitAmount + owner.OwnedInterest
		owner.Avaliable = 0
		Securities[i].Owners = append(Securities[i].Owners, owner)

		err := updateBankTotals(APIstub, args[0], Securities[i].SecurityID, Securities[i].Balance, false)
		if err != nil {
			return shim.Error(err.Error())
		}
		securityTotal.BankID = args[0]
		securityTotal.TotalBalance = Securities[i].Balance
		securityTotal.TotalInterest += owner.OwnedInterest
		securityTotal.DurationInterest = securityTotal.TotalInterest / int64(Securities[i].RepayPeriod)
		securityTotal.PaidDurationInterest = PaidDurationInterest
		securityTotal.CreateTime = TimeNow
		securityTotal.UpdateTime = TimeNow
		Securities[i].SecurityTotals = append(Securities[i].SecurityTotals, securityTotal)
		Securities[i].SecurityDurationDate = SecurityDurationDate
		Securities[i].TotalAmount = 25000 * unitAmount
		Securities[i].Balance = Securities[i].TotalAmount - owner.OwnedAmount
		SecurityAsBytes, _ := json.Marshal(Securities[i])
		//APIstub.PutState("Security"+strconv.Itoa(i), SecurityAsBytes)
		APIstub.PutState(Securities[i].SecurityID, SecurityAsBytes)
		fmt.Println("", Securities[i])
		//fmt.Println("daySub=", daySub(Securities[i].IssueDate, Securities[i].MaturityDate))
		i = i + 1
	}

	return shim.Success(nil)
}

//peer chaincode invoke -n mycc -c '{"Args":["createSecurity", "A07103","107A03","2018/03/02","2028/03/02","1","10","25000000000"]}' -C myc
func (s *SmartContract) createSecurity(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 7 {
		return shim.Error("Incorrect number of arguments. Expecting 7")
	}

	var newRepayPeriod int
	var newRate float64
	var newAmount int64
	newRate, err := strconv.ParseFloat(args[4], 64)
	if err != nil {
		return shim.Error(err.Error())
	}
	newRepayPeriod, err = strconv.Atoi(args[5])
	if err != nil {
		return shim.Error(err.Error())
	}
	newAmount, err = strconv.ParseInt(args[6], 10, 64)
	if err != nil {
		return shim.Error(err.Error())
	}

	var Security = Security{ObjectType: "security", SecurityID: args[0], SecurityName: args[1], IssueDate: args[2], MaturityDate: args[3], InterestRate: newRate, RepayPeriod: newRepayPeriod, TotalAmount: newAmount, Balance: newAmount}
	SecurityAsBytes, _ := json.Marshal(Security)
	err2 := APIstub.PutState(Security.SecurityID, SecurityAsBytes)
	if err2 != nil {
		return shim.Error("Failed to create state")
	}

	return shim.Success(nil)
}

//peer chaincode query -n mycc -c '{"Args":["querySecurity","A06101"]}' -C myc
func (s *SmartContract) querySecurity(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	SecurityAsBytes, _ := APIstub.GetState(args[0])
	return shim.Success(SecurityAsBytes)
}

//peer chaincode query -n mycc -c '{"Args":["queryAllSecurities","A000000","Z999999"]}' -C myc
func (s *SmartContract) queryAllSecurities(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

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
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode invoke -n mycc -c '{"Args":["changeSecurity", "A07103","107A03","2018/03/02","2028/03/02","1","10","25000000000","002000000001","002","1000000","0"]}' -C myc
func (s *SmartContract) changeSecurity(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 11 {
		return shim.Error("Incorrect number of arguments. Expecting 11")
	}

	TimeNow := time.Now().Format(timelayout)
	Today := SubString(TimeNow, 0, 8)

	var newRepayPeriod, newAvaliable int
	var newRate float64
	var newAmount, newOwnedAmount int64
	newRate, err := strconv.ParseFloat(args[4], 64)
	if err != nil {
		return shim.Error(err.Error())
	}
	newRepayPeriod, err = strconv.Atoi(args[5])
	if err != nil {
		return shim.Error(err.Error())
	}
	newAmount, err = strconv.ParseInt(args[6], 10, 64)
	if err != nil {
		return shim.Error(err.Error())
	}
	newOwnedAmount, err = strconv.ParseInt(args[9], 10, 64)
	if err != nil {
		return shim.Error(err.Error())
	}
	newAvaliable, err = strconv.Atoi(args[10])
	if err != nil {
		return shim.Error(err.Error())
	}

	SecurityAsBytes, _ := APIstub.GetState(args[0])
	Security := Security{}

	json.Unmarshal(SecurityAsBytes, &Security)
	Security.ObjectType = "security"
	Security.SecurityName = args[1]
	Security.IssueDate = args[2]
	Security.MaturityDate = args[3]
	Security.InterestRate = newRate
	Security.RepayPeriod = newRepayPeriod
	Security.TotalAmount = newAmount

	var doflg bool
	doflg = false
	var PaidDurationInterest int64
	PaidDurationInterest = 0

	var OwnedDurationDate []string
	var oldOwnedAmount int64
	var newBalance int64
	oldOwnedAmount = 0
	newBalance = 0
	var oldOwnedInterest int64
	var newOwnedInterest int64
	//var newInterest int64
	oldOwnedInterest = 0
	newOwnedInterest = 0
	//newInterest = 0

	for key, val := range Security.Owners {
		if val.OwnedAccountID == args[7] {
			oldOwnedAmount = Security.Owners[key].OwnedAmount
			oldOwnedInterest = Security.Owners[key].OwnedInterest
			Security.Balance += Security.Owners[key].OwnedAmount
			Security.Owners[key].OwnedBankID = args[8]
			Security.Owners[key].OwnedAmount = newOwnedAmount
			myOwnedAmount := float64(Security.Owners[key].OwnedAmount)
			OwnedInterest := perDayInterest * daySub(Security.IssueDate, Security.MaturityDate) * Security.InterestRate * myOwnedAmount
			Security.Owners[key].OwnedInterest = int64(OwnedInterest)
			newOwnedInterest = Security.Owners[key].OwnedInterest
			Security.Owners[key].OwnedDurationInterest = Security.Owners[key].OwnedInterest / int64(Security.RepayPeriod)
			j := 0
			var SecurityDurationDate []string

			for j < Security.RepayPeriod {
				NextPayInterestDate, _ := generateMaturity(Security.IssueDate, j+1, 0, 0)
				OwnedDurationDate = append(OwnedDurationDate, NextPayInterestDate)
				SecurityDurationDate = append(SecurityDurationDate, NextPayInterestDate)
				if Today >= NextPayInterestDate {
					PaidDurationInterest = newOwnedInterest * int64(j+1)
				}
				j = j + 1
			}
			Security.Owners[key].OwnedDurationDate = OwnedDurationDate
			Security.SecurityDurationDate = SecurityDurationDate
			Security.Owners[key].OwnedRepay = newOwnedAmount + Security.Owners[key].OwnedInterest
			Security.Owners[key].Avaliable = newAvaliable
			Security.Balance -= Security.Owners[key].OwnedAmount
			doflg = true
			break
		}
	}
	fmt.Printf("oldOwnedAmount: %d\n", oldOwnedAmount)
	if oldOwnedAmount > 0 {
		newBalance = newOwnedAmount - oldOwnedAmount
		//newInterest = newOwnedInterest - oldOwnedInterest
		fmt.Printf("newBalance: %d\n", newBalance)
		//err = updateBankTotals(APIstub, args[8], args[0], newBalance, newInterest, false)
		err = updateBankTotals(APIstub, args[8], args[0], newBalance, false)
	} else {
		//err = updateBankTotals(APIstub, args[8], args[0], newOwnedAmount, newOwnedInterest, false)
		err = updateBankTotals(APIstub, args[8], args[0], newOwnedAmount, false)
	}

	if err != nil {
		return shim.Error("Failed to change banktotal state")
	}

	if doflg != true {
		var owner Owner
		owner.OwnedAccountID = args[7]
		owner.OwnedBankID = args[8]
		owner.OwnedAmount = newOwnedAmount
		owner.OwnedInterest = int64(round(perDayMillionInterest*daySub(Security.IssueDate, Security.MaturityDate)*Security.InterestRate, 0)) * int64(newOwnedAmount/unitAmount)
		newOwnedInterest = owner.OwnedInterest
		owner.OwnedDurationInterest = owner.OwnedInterest / int64(Security.RepayPeriod)
		j := 0
		var SecurityDurationDate []string
		for j < Security.RepayPeriod {
			NextPayInterestDate, _ := generateMaturity(Security.IssueDate, j+1, 0, 0)
			owner.OwnedDurationDate = append(owner.OwnedDurationDate, NextPayInterestDate)
			SecurityDurationDate = append(SecurityDurationDate, NextPayInterestDate)
			if Today >= NextPayInterestDate {
				PaidDurationInterest = newOwnedInterest * int64(j+1)
			}
			j = j + 1
		}
		owner.OwnedRepay = newOwnedAmount + owner.OwnedInterest
		owner.Avaliable = newAvaliable
		Security.Owners = append(Security.Owners, owner)
		Security.SecurityDurationDate = SecurityDurationDate
		Security.Balance -= newOwnedAmount
	}

	doflg = false
	var securityTotal SecurityTotal
	BankID := args[8]

	fmt.Printf("BankID=%s\n", BankID)
	for key, val := range Security.SecurityTotals {
		fmt.Printf("1.Skey: %d\n", key)
		fmt.Printf("2.Sval: %s\n", val)
		if val.BankID == BankID {
			fmt.Printf("3.Skey: %d\n", key)
			fmt.Printf("4.Sval: %s\n", val)
			fmt.Printf("oldOwnedAmount: %d\n", oldOwnedAmount)
			fmt.Printf("oldOwnedInterest: %d\n", oldOwnedInterest)
			Security.SecurityTotals[key].TotalBalance -= oldOwnedAmount
			Security.SecurityTotals[key].TotalBalance += newOwnedAmount
			Security.SecurityTotals[key].TotalInterest -= oldOwnedInterest
			Security.SecurityTotals[key].TotalInterest += newOwnedInterest
			Security.SecurityTotals[key].DurationInterest = Security.SecurityTotals[key].TotalInterest / int64(Security.RepayPeriod)
			Security.SecurityTotals[key].PaidDurationInterest = PaidDurationInterest
			Security.SecurityTotals[key].UpdateTime = TimeNow
			doflg = true
			break
		}
	}
	if doflg != true {
		securityTotal.BankID = BankID
		securityTotal.TotalBalance = newOwnedAmount
		securityTotal.TotalInterest = newOwnedInterest
		securityTotal.DurationInterest = securityTotal.TotalInterest / int64(Security.RepayPeriod)
		securityTotal.PaidDurationInterest = PaidDurationInterest
		securityTotal.CreateTime = TimeNow
		securityTotal.UpdateTime = TimeNow
		Security.SecurityTotals = append(Security.SecurityTotals, securityTotal)
	}

	SecurityAsBytes, _ = json.Marshal(Security)
	err2 := APIstub.PutState(args[0], SecurityAsBytes)
	if err2 != nil {
		return shim.Error("Failed to change state")
	}

	return shim.Success(nil)
}

//peer chaincode invoke -n mycc -c '{"Args":["deleteSecurity", "A07103"]}' -C myc
// Deletes an entity from state
func (s *SmartContract) deleteSecurity(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	// Delete the key from the state in ledger
	err := APIstub.DelState(args[0])
	if err != nil {
		return shim.Error("Failed to delete state")
	}

	return shim.Success(nil)
}

//peer chaincode invoke -n mycc -c '{"Args":["deleteOwner", "A07103" , "002000000002"]}' -C myc
func (s *SmartContract) deleteOwner(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	SecurityAsBytes, _ := APIstub.GetState(args[0])
	Security := Security{}
	json.Unmarshal(SecurityAsBytes, &Security)

	var doflg bool
	doflg = false
	var owners []Owner
	filteredOwners := owners[:0]
	for key, val := range Security.Owners {
		if val.OwnedAccountID == args[1] {
			Security.Balance += Security.Owners[key].OwnedAmount
			doflg = true
		} else {
			filteredOwners = append(filteredOwners, val)
		}
	}

	if doflg == true {
		Security.Owners = filteredOwners
	}
	SecurityAsBytes, _ = json.Marshal(Security)
	err2 := APIstub.PutState(args[0], SecurityAsBytes)
	if err2 != nil {
		return shim.Error("Failed to delete state")
	}

	return shim.Success(nil)
}

//peer chaincode invoke -n mycc -c '{"Args":["changeSecurityStatus", "A06101" , "1" ]}' -C myc
func (s *SmartContract) changeSecurityStatus(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}
	var newStatus int
	newStatus, err := strconv.Atoi(args[1])
	if err != nil {
		return shim.Error(err.Error())
	}
	SecurityAsBytes, _ := APIstub.GetState(args[0])
	Security := Security{}
	json.Unmarshal(SecurityAsBytes, &Security)
	Security.SecurityStatus = newStatus
	SecurityAsBytes, _ = json.Marshal(Security)
	err2 := APIstub.PutState(args[0], SecurityAsBytes)
	if err2 != nil {
		return shim.Error("Failed to change state")
	}

	return shim.Success(nil)
}

func (s *SmartContract) changeOwnerAvaliable(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	SecurityAsBytes, _ := APIstub.GetState(args[0])
	Security := Security{}
	json.Unmarshal(SecurityAsBytes, &Security)

	var doflg bool
	var newAvaliable int
	newAvaliable, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error(err.Error())
	}
	doflg = false

	for key, val := range Security.Owners {
		if val.OwnedAccountID == args[1] {
			Security.Owners[key].Avaliable = newAvaliable
			doflg = true
		}
	}
	if doflg != true {
		return shim.Error("Failed to find ownedAccountID ")
	}

	SecurityAsBytes, _ = json.Marshal(Security)
	err2 := APIstub.PutState(args[0], SecurityAsBytes)
	if err2 != nil {
		return shim.Error("Failed to change state")
	}

	return shim.Success(nil)
}

//peer chaincode invoke -n mycc -c '{"Args":["updateOwnerInterest", "A07103"]}' -C myc
func (s *SmartContract) updateOwnerInterest(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	SecurityAsBytes, err := APIstub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	Security := Security{}
	json.Unmarshal(SecurityAsBytes, &Security)

	for key, _ := range Security.Owners {
		newOwnedAmount := float64(Security.Owners[key].OwnedAmount)
		OwnedInterest := perDayInterest * daySub(Security.IssueDate, Security.MaturityDate) * Security.InterestRate * newOwnedAmount
		Security.Owners[key].OwnedInterest = int64(OwnedInterest)
		Security.Owners[key].OwnedDurationInterest = Security.Owners[key].OwnedInterest / int64(Security.RepayPeriod)
		Security.Owners[key].OwnedRepay = Security.Owners[key].OwnedAmount + Security.Owners[key].OwnedInterest
	}

	SecurityAsBytes, _ = json.Marshal(Security)
	err2 := APIstub.PutState(args[0], SecurityAsBytes)
	if err2 != nil {
		return shim.Error("Failed to change state")
	}

	return shim.Success(SecurityAsBytes)
}

//peer chaincode query -n mycc -c '{"Args":["queryOwner","A07103"]}' -C myc
func (s *SmartContract) queryOwner(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	SecurityAsBytes, _ := APIstub.GetState(args[0])
	Security := Security{}
	json.Unmarshal(SecurityAsBytes, &Security)

	OwnerAsBytes, err := json.Marshal(Security.Owners)
	if err != nil {
		return shim.Error("Failed to query owner state")
	}

	return shim.Success(OwnerAsBytes)
}

//peer chaincode query -n mycc -c '{"Args":["queryOwnerLength","A07103"]}' -C myc
func (s *SmartContract) queryOwnerLength(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	SecurityAsBytes, _ := APIstub.GetState(args[0])
	Security := Security{}
	json.Unmarshal(SecurityAsBytes, &Security)

	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"SecurityID\":")
	buffer.WriteString("\"")
	buffer.WriteString(Security.SecurityID)
	buffer.WriteString("\"")
	buffer.WriteString(", \"OwnersLength\":")
	buffer.WriteString(strconv.Itoa(len(Security.Owners)))
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())

}

//peer chaincode query -n mycc -c '{"Args":["queryOwnerAccount","A07103","002000000001"]}' -C myc
func (s *SmartContract) queryOwnerAccount(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	SecurityAsBytes, _ := APIstub.GetState(args[0])
	Security := Security{}
	json.Unmarshal(SecurityAsBytes, &Security)

	var doflg bool
	doflg = false
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"SecurityID\":")
	buffer.WriteString("\"")
	buffer.WriteString(Security.SecurityID)
	buffer.WriteString("\"")

	for key, val := range Security.Owners {
		if val.OwnedAccountID == args[1] {
			buffer.WriteString(", \"OwnedKey\":")
			buffer.WriteString(strconv.Itoa(key))
			buffer.WriteString(", \"OwnedAccountID\":")
			buffer.WriteString(val.OwnedAccountID)
			buffer.WriteString(", \"OwnedBankID\":")
			buffer.WriteString(val.OwnedBankID)
			buffer.WriteString(", \"OwnedAmount\":")
			buffer.WriteString(strconv.FormatInt(val.OwnedAmount, 10))
			buffer.WriteString(", \"OwnedRepay\":")
			buffer.WriteString(strconv.FormatInt(val.OwnedRepay, 10))
			buffer.WriteString(", \"OwnedInterest\":")
			buffer.WriteString(strconv.FormatInt(val.OwnedInterest, 10))
			buffer.WriteString(", \"OwnedDurationInterest\":")
			buffer.WriteString(strconv.FormatInt(val.OwnedDurationInterest, 10))
			buffer.WriteString(", \"Avaliable\":")
			buffer.WriteString(strconv.Itoa(val.Avaliable))
			doflg = true
		}
	}
	if doflg != true {
		return shim.Error("Failed to find ownedAccountID ")
	}
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc -c '{"Args":["querySecurityStatus","A07103"]}' -C myc
func (s *SmartContract) querySecurityStatus(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	SecurityAsBytes, _ := APIstub.GetState(args[0])
	Security := Security{}
	json.Unmarshal(SecurityAsBytes, &Security)

	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"SecurityID\":")
	buffer.WriteString("\"")
	buffer.WriteString(Security.SecurityID)
	buffer.WriteString("\"")
	buffer.WriteString(", \"SecurityStatus\":")
	buffer.WriteString(strconv.Itoa(Security.SecurityStatus))
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())

}

func getSecurityStructFromID(
	stub shim.ChaincodeStubInterface,
	SecurityID string) (*Security, error) {

	var errMsg string
	security := &Security{}
	securityAsBytes, err := stub.GetState(SecurityID)
	if err != nil {
		return security, err
	} else if securityAsBytes == nil {
		errMsg = fmt.Sprintf("Error: SecurityID does not exist (%s)", SecurityID)
		return security, errors.New(errMsg)
	}
	err = json.Unmarshal([]byte(securityAsBytes), security)
	if err != nil {
		return security, err
	}
	return security, nil
}

//peer chaincode query -n mycc -c '{"Args":["getHistoryForSecurity","A06101"]}' -C myc
func (s *SmartContract) getHistoryForSecurity(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	SecurityID := args[0]

	fmt.Printf("- start getHistoryForSecurity: %s\n", SecurityID)

	resultsIterator, err := APIstub.GetHistoryForKey(SecurityID)
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

	fmt.Printf("- getHistoryForSecurity returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

func (s *SmartContract) queryAllSecurityKeys(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

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

//updateSecurityTotalAmount(stub,"A07106","002","004",1000)
func updateSecurityTotalAmount(stub shim.ChaincodeStubInterface, SecurityID string, BankFrom string, BankTo string, Payment int64, isNegative bool) (*Security, error) {
	fmt.Printf("updateSecurityTotalAmount: SecurityID=%s,BankFrom=%s,BankTo=%s,Payment=%d\n", SecurityID, BankFrom, BankTo, Payment)
	TimeNow := time.Now().Format(timelayout)
	Today := SubString(TimeNow, 0, 8)
	newSecurityID := SecurityID
	senderBank := BankFrom
	receiverBank := BankTo
	Security, err := getSecurityStructFromID(stub, newSecurityID)
	if err != nil {
		return Security, err
	}
	var doflg bool
	doflg = false
	var securityTotal SecurityTotal
	newOwnedAmount := float64(Payment)
	OwnedInterest := perDayInterest * daySub(Security.IssueDate, Security.MaturityDate) * Security.InterestRate * newOwnedAmount
	Interest := int64(OwnedInterest)
	DurationInterest := Interest / int64(Security.RepayPeriod)
	var PaidDurationPeriod int64
	PaidDurationPeriod = 0
	j := 0
	for j < Security.RepayPeriod {
		NextPayInterestDate, _ := generateMaturity(Security.IssueDate, j+1, 0, 0)
		if Today >= NextPayInterestDate {
			PaidDurationPeriod = int64(j + 1)
		}
		j = j + 1
	}

	PaidDurationInterest := DurationInterest * PaidDurationPeriod
	fmt.Printf("DurationInterest: %d\n", DurationInterest)
	fmt.Printf("PaidDurationPeriod: %d\n", PaidDurationPeriod)
	fmt.Printf("PaidDurationInterest: %d\n", PaidDurationInterest)

	for key, val := range Security.SecurityTotals {
		fmt.Printf("1.Skey: %d\n", key)
		fmt.Printf("2.Sval: %s\n", val)

		if val.BankID == senderBank {
			fmt.Printf("3.Skey: %d\n", key)
			fmt.Printf("4.Sval: %s\n", val)
			if isNegative == false {
				Security.SecurityTotals[key].TotalBalance += Payment
				Security.SecurityTotals[key].DurationInterest += DurationInterest
				Security.SecurityTotals[key].PaidDurationInterest += PaidDurationInterest
			} else {
				Security.SecurityTotals[key].TotalBalance -= Payment
				Security.SecurityTotals[key].DurationInterest -= DurationInterest
				Security.SecurityTotals[key].PaidDurationInterest -= PaidDurationInterest
			}
			Security.SecurityTotals[key].UpdateTime = TimeNow
			doflg = true
		}
		if val.BankID == receiverBank {
			fmt.Printf("5.Skey: %d\n", key)
			fmt.Printf("6.Sval: %s\n", val)
			if isNegative == false {
				Security.SecurityTotals[key].TotalBalance -= Payment
				Security.SecurityTotals[key].DurationInterest -= DurationInterest
				Security.SecurityTotals[key].PaidDurationInterest -= PaidDurationInterest
			} else {
				Security.SecurityTotals[key].TotalBalance += Payment
				Security.SecurityTotals[key].DurationInterest += DurationInterest
				Security.SecurityTotals[key].PaidDurationInterest += PaidDurationInterest
			}
			Security.SecurityTotals[key].UpdateTime = TimeNow
			doflg = true
		}

	}
	if doflg != true {
		securityTotal.BankID = senderBank
		securityTotal.TotalBalance = Payment
		securityTotal.TotalInterest = Interest
		securityTotal.DurationInterest = DurationInterest
		securityTotal.PaidDurationInterest = PaidDurationInterest
		securityTotal.CreateTime = TimeNow
		securityTotal.UpdateTime = TimeNow
		Security.SecurityTotals = append(Security.SecurityTotals, securityTotal)

		securityTotal.BankID = receiverBank
		securityTotal.TotalBalance = Payment
		securityTotal.TotalInterest = Interest
		securityTotal.DurationInterest = DurationInterest
		securityTotal.PaidDurationInterest = PaidDurationInterest
		securityTotal.CreateTime = TimeNow
		securityTotal.UpdateTime = TimeNow
		Security.SecurityTotals = append(Security.SecurityTotals, securityTotal)
	}

	for key, _ := range Security.Owners {
		newOwnedAmount := float64(Security.Owners[key].OwnedAmount)
		OwnedInterest := perDayInterest * daySub(Security.IssueDate, Security.MaturityDate) * Security.InterestRate * newOwnedAmount
		Security.Owners[key].OwnedInterest = int64(OwnedInterest)
		Security.Owners[key].OwnedDurationInterest = Security.Owners[key].OwnedInterest / int64(Security.RepayPeriod)
		Security.Owners[key].OwnedRepay = Security.Owners[key].OwnedAmount + Security.Owners[key].OwnedInterest
	}
	/*
		securityAsBytes, err := json.Marshal(security)
		if err != nil {
			return security,err
		}
		err = stub.PutState(newSecurityID, securityAsBytes)
		if err != nil {
			return security,err
		}
	*/
	return Security, nil
}

//peer chaincode invoke -n mycc -c '{"Args":["updateSecurityTotals", "A07106","20190701"]}' -C myc
func (s *SmartContract) updateSecurityTotals(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) < 2 {
		return shim.Error("Keys operation must include 2 arguments")
	}
	SecurityID := args[0]
	BaselineDay := args[1]
	fmt.Printf("updateSecurityTotals: SecurityID=%s,BaselineDay=%s\n", SecurityID, BaselineDay)
	TimeNow := time.Now().Format(timelayout)
	Today := SubString(TimeNow, 0, 8)
	if BaselineDay != "" {
		Today = BaselineDay
	}
	fmt.Printf("Today=%s\n", Today)
	newSecurityID := SecurityID
	security, err := getSecurityStructFromID(stub, newSecurityID)
	fmt.Printf("new newSecurityID=%s\n", newSecurityID)
	if err != nil {
		return shim.Error(err.Error())
	}

	var PaidDurationPeriod int64
	PaidDurationPeriod = 0
	var TotalBalance, TotalInterest, DurationInterest, PaidDurationInterest int64
	TotalBalance = 0
	TotalInterest = 0
	j := 0
	for j < security.RepayPeriod {
		NextPayInterestDate, _ := generateMaturity(security.IssueDate, j+1, 0, 0)
		if Today >= NextPayInterestDate {
			PaidDurationPeriod = int64(j + 1)
		}
		j = j + 1
	}

	for key, _ := range security.Owners {
		newOwnedAmount := float64(security.Owners[key].OwnedAmount)
		TotalBalance += int64(newOwnedAmount)
		OwnedInterest := perDayInterest * daySub(security.IssueDate, security.MaturityDate) * security.InterestRate * newOwnedAmount
		TotalInterest += int64(OwnedInterest)
		security.Owners[key].OwnedInterest = int64(OwnedInterest)
		security.Owners[key].OwnedDurationInterest = security.Owners[key].OwnedInterest / int64(security.RepayPeriod)
		security.Owners[key].OwnedRepay = security.Owners[key].OwnedAmount + security.Owners[key].OwnedInterest
	}

	for key, val := range security.SecurityTotals {
		fmt.Printf("1.Skey: %d\n", key)
		fmt.Printf("2.Sval: %s\n", val)

		Interest := int64(TotalInterest)
		DurationInterest = Interest / int64(security.RepayPeriod)
		PaidDurationInterest = DurationInterest * PaidDurationPeriod
		fmt.Printf("Interest: %d\n", Interest)
		fmt.Printf("DurationInterest: %d\n", DurationInterest)
		fmt.Printf("PaidDurationPeriod: %d\n", PaidDurationPeriod)
		fmt.Printf("PaidDurationInterest: %d\n", PaidDurationInterest)
		security.SecurityTotals[key].TotalBalance = TotalBalance
		security.SecurityTotals[key].TotalInterest = TotalInterest
		security.SecurityTotals[key].DurationInterest = DurationInterest
		security.SecurityTotals[key].PaidDurationInterest = PaidDurationInterest
		security.SecurityTotals[key].UpdateTime = TimeNow

	}

	securityAsBytes, err := json.Marshal(security)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(newSecurityID, securityAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

// The main function is only relevant in unit test mode. Only included here for completeness.
func main() {

	// Create a new Smart Contract
	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating new Smart Contract: %s", err)
	}
}

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(msInt/millisPerSecond,
		(msInt%millisPerSecond)*nanosPerMillisecond), nil
}

func generateMaturity(issueDate string, years int, months int, days int) (string, error) {

	t, err := msToTime(makeTimestamp(issueDate))
	if err != nil {
		return "", err
	}

	maturityDate := t.AddDate(years, months, days)
	sDate := maturityDate.Format(layout)
	return sDate, nil

}

func makeTimestamp(aDate string) string {
	var stamp int64
	t, _ := time.Parse(layout, aDate)
	stamp = t.UnixNano() / int64(time.Millisecond)
	str := strconv.FormatInt(stamp, 10)
	return str
}

func getDateUnix(mydate string) int64 {
	tm2, _ := time.Parse(layout, mydate)
	return tm2.Unix()
}

func daySub(d1, d2 string) float64 {
	t1, _ := time.Parse(layout, d1)
	t2, _ := time.Parse(layout, d2)
	return float64(timeSub(t2, t1))
}

func timeSub(t1, t2 time.Time) int {
	t1 = t1.UTC().Truncate(24 * time.Hour)
	t2 = t2.UTC().Truncate(24 * time.Hour)
	return int(t1.Sub(t2).Hours() / 24)
}

func SubString(str string, begin, length int) (substr string) {
	// 將字串轉成[]rune
	rs := []rune(str)
	lth := len(rs)

	// 範圍判断
	if begin < 0 {
		begin = 0
	}
	if begin >= lth {
		begin = lth
	}
	end := begin + length
	if end > lth {
		end = lth
	}

	// 返回字串
	return string(rs[begin:end])
}

func UnicodeIndex(str, substr string) int {
	// 子字串在字串的位置
	result := strings.Index(str, substr)
	if result >= 0 {
		// 取得子字串之前的字串並轉換成[]byte
		prefix := []byte(str)[0:result]
		// 將字串轉換成[]rune
		rs := []rune(string(prefix))
		// 取得rs的長度，即子字串在字串的位置
		result = len(rs)
	}

	return result
}

func round(v float64, decimals int) float64 {
	var pow float64 = 1
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int((v*pow)+0.5)) / pow
}

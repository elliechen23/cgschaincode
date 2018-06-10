package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

const TransactionObjectType string = "Transaction"
const QueuedTXObjectType string = "QueuedTX"
const HistoryTXObjectType string = "HistoryTX"
const timelayout string = "20060102150405"

//同資flag
const approved0 string = "0"   //預設款夠，Finished
const approved1 string = "1"   //等待，Waiting4Payment
const approved2 string = "2"   //不夠，Waiting4Payment
const approved21 string = "21" //款不足，Cancelled
const approved22 string = "22" //放行，Finished
const approved3 string = "3"   //系統錯誤，Cancelled

type Transaction struct {
	ObjectType     string `json:"docType"`        // default set to "Transaction"
	TXID           string `json:"TXID"`           // Transaction ID
	TXType         string `json:"TXType"`         // Transaction TXType BUY or SELL
	TXFrom         string `json:"TXFrom"`         // Transaction from
	TXTo           string `json:"TXTo"`           // Transaction to
	BankFrom       string `json:"BankFrom"`       // Bank from
	BankTo         string `json:"BankTo"`         // Bank to
	SecurityID     string `json:"SecurityID"`     // SecurityID
	SecurityAmount int64  `json:"SecurityAmount"` // SecurityAmount
	Payment        int64  `json:"Payment"`        // Payment
	isPutToQueue   bool   `json:"isPutToQueue"`   // isPutToQueue
	TXStatus       string `json:"TXStatus"`       // Pending, Matched, Finished, Cancelled, PaymentError,
	IsFrozen       bool   `json:"isFrozen"`
	CreateTime     string `json:"createTime"`
	UpdateTime     string `json:"updateTime"`
	TXIndex        string `json:"TXIndex"`  // Transaction Index(全部比對)
	TXSIndex       string `json:"TXSIndex"` // Transaction Short Index(沒有比對SecurityAmount,Payment)
	TXHcode        string `json:"TXHcode"`  // Transaction Hcode(更正交易序號)
	TXFromBalance  int64  `json:"TXFromBalance"`
	TXFromPosition int64  `json:"TXFromPosition"`
	MatchedTXID    string `json:"MatchedTXID"`
	TXMemo         string `json:"TXMemo"`   //交易說明
	TXErrMsg       string `json:"TXErrMsg"` //交易錯誤說明
}

/*
TXData1 = BankFrom + TXFrom + BankTo + TXTo + SecurityID + strconv.FormatInt(SecurityAmount, 10) + strconv.FormatInt(Payment, 10)
TXIndex = getSHA256(TXData1)
TXData2 = BankFrom + TXFrom + BankTo + TXTo + SecurityID
TXSIndex = getSHA256(TXData2)
*/

type QueuedTransaction struct {
	ObjectType   string        `json:"docType"` // default set to "QueuedTX"
	TXKEY        string        `json:"TXKEY"`   //TXDATE(YYYYMMDD)
	TXIDs        []string      `json:"TXIDs"`
	TXIndexs     []string      `json:"TXIndexs"`
	TXSIndexs    []string      `json:"TXSIndexs"`
	Transactions []Transaction `json:"Transactions"`
}

type TransactionHistory struct {
	ObjectType   string        `json:"docType"` // default set to "HistoryTX"
	TXKEY        string        `json:"TXKEY"`   //TXDATE(HYYYYMMDD)
	TXIDs        []string      `json:"TXIDs"`
	TXIndexs     []string      `json:"TXIndexs"`
	TXSIndexs    []string      `json:"TXSIndexs"`
	TXStatus     []string      `json:"TXStatus"`
	TXKinds      []string      `json:"TXKinds"`
	Transactions []Transaction `json:"Transactions"`
}

/*
peer chaincode invoke -n mycc3 -c '{"Args":["submitApproveTransaction", "BANK004B00400000000120180415070724","0","BANKCBC"]}' -C myc -v 9.0
peer chaincode invoke -n mycc3 -c '{"Args":["submitApproveTransaction", "BANK002S00200000000120180415065316","0","BANKCBC"]}' -C myc -v 9.0

*/

func (s *SmartContract) submitApproveTransaction(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	TimeNow := time.Now().Format(timelayout)
	err := checkArgArrayLength(args, 2)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(args[0]) <= 0 {
		return shim.Error("TXID must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("Admin must be a non-empty string")
	}

	TXID := strings.ToUpper(args[0])
	TXKEY := SubString(TimeNow, 0, 8)
	HTXKEY := "H" + SubString(TimeNow, 0, 8)

	ApproveFlag := approved0
	ValueAsBytes, err := stub.GetState("approveflag")
	if err == nil {
		ApproveFlag = string(ValueAsBytes)
	}
	fmt.Printf("1.ApproveFlag=%s\n", ApproveFlag)

	isApproved := true
	NewStatus := "Finished"
	if ApproveFlag == approved0 {
		isApproved = true
		NewStatus = "Finished"
	} else if ApproveFlag == approved1 {
		isApproved = true
		NewStatus = "Waiting4Payment"
	} else if ApproveFlag == approved2 {
		isApproved = false
		NewStatus = "PaymentError"
	} else if ApproveFlag == approved21 {
		isApproved = false
		NewStatus = "Cancalled"
	} else if ApproveFlag == approved21 {
		isApproved = true
		NewStatus = "Finished"
	} else if ApproveFlag == approved3 {
		isApproved = false
		NewStatus = "Cancalled"
	}

	transaction, err := getTransactionStructFromID(stub, TXID)
	if err != nil {
		return shim.Error("TXID transacton does not found.")
	}
	MatchedTXID := transaction.MatchedTXID
	SecurityID := transaction.SecurityID
	SecurityAmount := transaction.SecurityAmount
	Payment := transaction.Payment
	TXType := transaction.TXType
	TXFrom := transaction.TXFrom
	TXTo := transaction.TXTo

	fmt.Printf("1.Approved TXID=%s\n", TXID)
	fmt.Printf("2.Approved MatchedTXID=%s\n", MatchedTXID)
	fmt.Printf("3.Approved TXKEY=%s\n", TXKEY)
	fmt.Printf("4.Approved TXKEY=%s\n", HTXKEY)

	if isApproved != true {
		err := updateQueuedTransactionApproveStatus(stub, TXKEY, TXID, MatchedTXID, NewStatus)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = updateHistoryTransactionApproveStatus(stub, HTXKEY, TXID, MatchedTXID, NewStatus)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = updateTransactionStatus(stub, TXID, NewStatus, "")
		if err != nil {
			return shim.Error(err.Error())
		}
		err = updateTransactionStatus(stub, MatchedTXID, NewStatus, "")
		if err != nil {
			return shim.Error(err.Error())
		}

		if TXType == "S" {
			senderBalance, receiverBalance, err := resetAccountBalance(stub, SecurityID, SecurityAmount, Payment, TXFrom, TXTo)
			senderBalance, receiverBalance, err = resetSecurityAmount(stub, SecurityID, Payment, TXFrom, TXTo)
			err = updateBankTotals(stub, TXFrom, SecurityID, Payment, false)
			if err != nil {
				return shim.Error(err.Error())
			}
			err = updateBankTotals(stub, TXTo, SecurityID, Payment, true)
			if err != nil {
				return shim.Error(err.Error())
			}
			if (senderBalance < 0) || (receiverBalance < 0) {
				return shim.Error("senderBalance,receiverBalance <0")
			}

		}
		if TXType == "B" {
			senderBalance, receiverBalance, err := resetAccountBalance(stub, SecurityID, SecurityAmount, Payment, TXTo, TXFrom)
			senderBalance, receiverBalance, err = resetSecurityAmount(stub, SecurityID, Payment, TXTo, TXFrom)
			err = updateBankTotals(stub, TXFrom, SecurityID, Payment, false)
			if err != nil {
				return shim.Error(err.Error())
			}
			err = updateBankTotals(stub, TXTo, SecurityID, Payment, true)
			if err != nil {
				return shim.Error(err.Error())
			}
			if (senderBalance < 0) || (receiverBalance < 0) {
				return shim.Error("senderBalance,receiverBalance <0")
			}

		}

	} else if isApproved == true {
		fmt.Printf("5.Approved TXID=%s\n", TXID)
		fmt.Printf("6.Approved MatchedTXID=%s\n", MatchedTXID)
		fmt.Printf("7.Approved TXKEY=%s\n", TXKEY)
		fmt.Printf("8.Approved TXKEY=%s\n", HTXKEY)
		err := updateQueuedTransactionApproveStatus(stub, TXKEY, TXID, MatchedTXID, NewStatus)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = updateHistoryTransactionApproveStatus(stub, HTXKEY, TXID, MatchedTXID, NewStatus)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = updateTransactionStatus(stub, TXID, NewStatus, MatchedTXID)
		if err != nil {
			return shim.Error(err.Error())
		}

		err = updateTransactionStatus(stub, MatchedTXID, NewStatus, TXID)
		if err != nil {
			return shim.Error(err.Error())
		}

	}

	return shim.Success(nil)

}

//peer chaincode invoke -n mycc3 -c '{"Args":["submitEndDayTransaction", "BANK004S00400000000120180414121032" , "BANKCBC" ]}' -C myc -v 1.0

func (s *SmartContract) submitEndDayTransaction(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {
	//var MatchedTXID string
	//MatchedTXID = ""
	TimeNow := time.Now().Format(timelayout)
	err := checkArgArrayLength(args, 2)
	if err != nil {
		return shim.Error(err.Error())
	}

	if len(args[0]) <= 0 {
		return shim.Error("TXID  must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return shim.Error("Admin  must be a non-empty string")
	}

	TXID := strings.ToUpper(args[0])
	//Admin := strings.ToUpper(args[1])
	//TimeNow := time.Now().Format(timelayout)
	TXKEY := SubString(TimeNow, 0, 8)
	HTXKEY := "H" + SubString(TimeNow, 0, 8)

	MatchedTXID, err2 := updateEndDayTransactionStatus(stub, TXID)
	if err2 != nil {
		return shim.Error(err2.Error())
	}

	err = updateEndDayQueuedTransactionStatus(stub, TXKEY, TXID, MatchedTXID)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = updateEndDayHistoryTransactionStatus(stub, HTXKEY, TXID, MatchedTXID)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)

}

/*
peer chaincode invoke -n mycc3 -c '{"Args":["securityTransfer", "B","004000000001" , "002000000001" , "A07103" , "102000","100000","true"]}' -C myc -v 1.0
peer chaincode invoke -n mycc3 -c '{"Args":["securityTransfer", "S","002000000001" , "004000000001" , "A07103" , "102000","100000","true"]}' -C myc -v 1.0

peer chaincode invoke -n mycc3 -c '{"Args":["securityTransfer", "B","004000000001" , "004000000002" , "A07103" , "102000","100000","true"]}' -C myc
peer chaincode invoke -n mycc3 -c '{"Args":["securityTransfer", "S","004000000002" , "004000000001" , "A07103" , "102000","100000","true"]}' -C myc

peer chaincode invoke -n mycc3 -c '{"Args":["securityTransfer", "S","002000000001" , "002000000002" , "A07103" , "102000","100000","true"]}' -C myc
peer chaincode invoke -n mycc3 -c '{"Args":["securityTransfer", "B","002000000002" , "002000000001" , "A07103" , "102000","100000","true"]}' -C myc


*/
func (s *SmartContract) securityTransfer(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	TimeNow := time.Now().Format(timelayout)
	newTX, isPutInQueue, errMsg := validateTransaction(stub, args)
	if errMsg != "" {
		//return shim.Error(err.Error())
		newTX.TXErrMsg = errMsg
	}
	TXIndex := newTX.TXIndex
	TXSIndex := newTX.TXSIndex
	TXID := newTX.TXID
	TXType := newTX.TXType
	SecurityID := newTX.SecurityID
	TXFrom := newTX.TXFrom
	TXTo := newTX.TXTo
	BankFrom := newTX.BankFrom
	BankTo := newTX.BankTo
	Payment := newTX.Payment
	SecurityAmount := newTX.SecurityAmount
	TXStatus := newTX.TXStatus

	var doflg bool
	var TXKinds string
	doflg = false
	TXKEY := SubString(TimeNow, 0, 8) //A0710220180326
	HTXKEY := "H" + TXKEY
	if BankFrom != BankTo {
		if SecurityAmount == 0 {
			if TXType == "S" {
				TXKinds = "跨行FOP轉出"
			} else {
				TXKinds = "跨行FOP轉入"
			}
		} else {
			if TXType == "S" {
				TXKinds = "跨行DVP轉出"
			} else {
				TXKinds = "跨行DVP轉入"
			}
		}
	} else {
		if TXType == "S" {
			TXKinds = "自行FOP轉出"
		} else {
			TXKinds = "自行FOP轉入"
		}
	}
	ApproveFlag := approved0
	ValueAsBytes, err := stub.GetState("approveflag")
	if err == nil {
		ApproveFlag = string(ValueAsBytes)
	}

	fmt.Printf("2.ApproveFlag=%s\n", ApproveFlag)

	if isPutInQueue == true {

		queueAsBytes, err := stub.GetState(TXKEY)
		if err != nil {
			//return shim.Error(err.Error())
			newTX.TXErrMsg = TXKEY + ":QueueID does not exits."
		}
		queuedTx := QueuedTransaction{}
		json.Unmarshal(queueAsBytes, &queuedTx)

		historyAsBytes, err := stub.GetState(HTXKEY)
		if err != nil {
			newTX.TXErrMsg = HTXKEY + ":HistoryID does not exits."
		}
		historyNewTX := TransactionHistory{}
		json.Unmarshal(historyAsBytes, &historyNewTX)

		if queueAsBytes == nil {
			queuedTx.ObjectType = QueuedTXObjectType
			queuedTx.TXKEY = TXKEY
			queuedTx.Transactions = append(queuedTx.Transactions, newTX)
			queuedTx.TXIndexs = append(queuedTx.TXIndexs, TXIndex)
			queuedTx.TXSIndexs = append(queuedTx.TXSIndexs, TXSIndex)
			queuedTx.TXIDs = append(queuedTx.TXIDs, TXID)
			if historyAsBytes == nil {
				historyNewTX.ObjectType = HistoryTXObjectType
				historyNewTX.TXKEY = HTXKEY
				historyNewTX.Transactions = append(historyNewTX.Transactions, newTX)
				historyNewTX.TXIndexs = append(historyNewTX.TXIndexs, TXIndex)
				historyNewTX.TXSIndexs = append(historyNewTX.TXSIndexs, TXSIndex)
				historyNewTX.TXIDs = append(historyNewTX.TXIDs, TXID)
				historyNewTX.TXStatus = append(historyNewTX.TXStatus, newTX.TXStatus)
				historyNewTX.TXKinds = append(historyNewTX.TXKinds, TXKinds)
			}
		} else if queueAsBytes != nil {
			for key, val := range queuedTx.Transactions {
				if val.TXIndex == TXIndex && val.TXStatus == TXStatus && val.TXFrom != TXFrom && val.TXType != TXType && val.TXID != TXID {
					fmt.Println("1.TXIndex= " + TXIndex + "\n")
					fmt.Println("2.TXFrom= " + TXFrom + "\n")
					fmt.Println("3.TXType= " + TXType + "\n")
					fmt.Println("4.TXID= " + TXID + "\n")
					fmt.Println("5.val.TXID= " + val.TXID + "\n")
					fmt.Println("6.TXStatus= " + TXStatus + "\n")
					fmt.Println("7.val.TXStatus= " + val.TXStatus + "\n")

					if TXStatus == "Pending" && val.TXStatus == "Pending" {
						if doflg == true {
							//return shim.Error("doflg eq to true")
							newTX.TXErrMsg = "doflg can not equle to true."
						}
						newTX.MatchedTXID = val.TXID
						queuedTx.Transactions[key].MatchedTXID = TXID
						historyNewTX.Transactions[key].MatchedTXID = TXID
						err = updateTransactionStatus(stub, val.TXID, "Matched", TXID)
						if err != nil {
							//return shim.Error(err.Error())
							newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Matched."
						}
						queuedTx.Transactions[key].TXStatus = "Matched"
						historyNewTX.Transactions[key].TXStatus = "Matched"
						historyNewTX.TXStatus[key] = "Matched"
						newTX.TXStatus = "Matched"
						queuedTx.Transactions[key].TXMemo = ""
						historyNewTX.Transactions[key].TXMemo = ""
						newTX.TXMemo = ""
						if TXType == "S" {
							//轉出          轉入
							senderBalance, receiverBalance, err := updateAccountBalance(stub, SecurityID, SecurityAmount, Payment, TXFrom, TXTo)
							senderBalance, receiverBalance, err = updateSecurityAmount(stub, SecurityID, Payment, TXFrom, TXTo)
							err = updateBankTotals(stub, TXFrom, SecurityID, Payment, true)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateBankTotals TXFrom:" + TXFrom
							}
							err = updateBankTotals(stub, TXTo, SecurityID, Payment, false)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateBankTotals2 TXTo:" + TXTo
							}
							if (senderBalance < 0) || (receiverBalance < 0) {
								//return shim.Error("TxType=S - senderBalance or receiverBalance <0")
								newTX.TXErrMsg = "TxType=S - senderBalance or receiverBalance <0"
							}

						}
						if TXType == "B" {
							//轉出          轉入
							senderBalance, receiverBalance, err := updateAccountBalance(stub, SecurityID, SecurityAmount, Payment, TXTo, TXFrom)
							senderBalance, receiverBalance, err = updateSecurityAmount(stub, SecurityID, Payment, TXTo, TXFrom)
							err = updateBankTotals(stub, TXFrom, SecurityID, Payment, true)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateBankTotals TXFrom:" + TXFrom
							}
							err = updateBankTotals(stub, TXTo, SecurityID, Payment, false)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateBankTotals2 TXTo:" + TXTo
							}
							if (senderBalance < 0) || (receiverBalance < 0) {
								//return shim.Error("TxType=B - senderBalance or receiverBalance <0")
								newTX.TXErrMsg = "TxType=B - senderBalance or receiverBalance <0"
							}

						}
						newTX.IsFrozen = true
						queuedTx.Transactions[key].IsFrozen = true
						historyNewTX.Transactions[key].IsFrozen = true
						if BankFrom != BankTo {
							if SecurityAmount != 0 {
								if ApproveFlag == approved0 {
									queuedTx.Transactions[key].TXStatus = "Finished"
									historyNewTX.Transactions[key].TXStatus = "Finished"
									historyNewTX.TXStatus[key] = "Finished"
									newTX.TXStatus = "Finished"
									err := updateTransactionStatus(stub, val.TXID, "Finished", TXID)
									if err != nil {
										//return shim.Error(err.Error())
										newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Finished."
									}
								} else {
									queuedTx.Transactions[key].TXStatus = "Waiting4Payment"
									historyNewTX.Transactions[key].TXStatus = "Waiting4Payment"
									historyNewTX.TXStatus[key] = "Waiting4Payment"
									newTX.TXStatus = "Waiting4Payment"
									err := updateTransactionStatus(stub, val.TXID, "Waiting4Payment", TXID)
									if err != nil {
										//return shim.Error(err.Error())
										newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Waiting4Payment."
									}
								}
							} else {
								queuedTx.Transactions[key].TXStatus = "Finished"
								historyNewTX.Transactions[key].TXStatus = "Finished"
								historyNewTX.TXStatus[key] = "Finished"
								newTX.TXStatus = "Finished"
								err := updateTransactionStatus(stub, val.TXID, "Finished", TXID)
								if err != nil {
									//return shim.Error(err.Error())
									newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Finished."
								}
							}
						} else {
							queuedTx.Transactions[key].TXStatus = "Finished"
							historyNewTX.Transactions[key].TXStatus = "Finished"
							historyNewTX.TXStatus[key] = "Finished"
							newTX.TXStatus = "Finished"
							err := updateTransactionStatus(stub, val.TXID, "Finished", TXID)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Finished."
							}
						}

						doflg = true
						break
					}
				} else {
					fmt.Println("1.TXSIndex= " + TXSIndex + "\n")
					if val.TXSIndex == TXSIndex && val.TXStatus == TXStatus && val.TXIndex != TXIndex && val.TXFrom != TXFrom && val.TXType != TXType && val.TXID != TXID {
						if TXStatus == "Pending" && val.TXStatus == "Pending" {
							if SecurityAmount != val.SecurityAmount {
								newTX.MatchedTXID = val.TXID
								queuedTx.Transactions[key].MatchedTXID = TXID
								historyNewTX.Transactions[key].MatchedTXID = TXID
								newTX.TXMemo = "交易金額疑似有誤"
								queuedTx.Transactions[key].TXMemo = "交易金額疑似有誤"
								historyNewTX.Transactions[key].TXMemo = "交易金額疑似有誤"
								newTX.TXErrMsg = "SecurityAmount != val.SecurityAmount"
								queuedTx.Transactions[key].TXErrMsg = "SecurityAmount != val.SecurityAmount"
								historyNewTX.Transactions[key].TXErrMsg = "SecurityAmount != val.SecurityAmount"
							} else if Payment != val.Payment {
								newTX.MatchedTXID = val.TXID
								queuedTx.Transactions[key].MatchedTXID = TXID
								historyNewTX.Transactions[key].MatchedTXID = TXID
								newTX.TXMemo = "交易面額疑似有誤"
								queuedTx.Transactions[key].TXMemo = "交易面額疑似有誤"
								historyNewTX.Transactions[key].TXMemo = "交易面額疑似有誤"
								newTX.TXErrMsg = "Payment != val.Payment"
								queuedTx.Transactions[key].TXErrMsg = "Payment != val.Payment"
								historyNewTX.Transactions[key].TXErrMsg = "Payment != val.Payment"
							}
						}
					}
				}
			}
		}
		if queueAsBytes != nil {
			if historyAsBytes != nil {
				queuedTx.ObjectType = QueuedTXObjectType
				queuedTx.TXKEY = TXKEY
				queuedTx.Transactions = append(queuedTx.Transactions, newTX)
				queuedTx.TXIndexs = append(queuedTx.TXIndexs, TXIndex)
				queuedTx.TXIDs = append(queuedTx.TXIDs, TXID)

				historyNewTX.ObjectType = HistoryTXObjectType
				historyNewTX.TXKEY = HTXKEY
				historyNewTX.Transactions = append(historyNewTX.Transactions, newTX)
				historyNewTX.TXIndexs = append(historyNewTX.TXIndexs, TXIndex)
				historyNewTX.TXIDs = append(historyNewTX.TXIDs, TXID)
				historyNewTX.TXStatus = append(historyNewTX.TXStatus, newTX.TXStatus)
				historyNewTX.TXKinds = append(historyNewTX.TXKinds, TXKinds)
			}
		}
		QueuedAsBytes, err := json.Marshal(queuedTx)
		err = stub.PutState(TXKEY, QueuedAsBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
		historyAsBytes, err = json.Marshal(historyNewTX)
		err = stub.PutState(HTXKEY, historyAsBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	TransactionAsBytes, err := json.Marshal(newTX)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(TXID, TransactionAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func validateTransaction(
	stub shim.ChaincodeStubInterface,
	args []string) (Transaction, bool, string) {
	TimeNow := time.Now().Format(timelayout)
	var err error
	var TXData1, TXData2, TXIndex, TXSIndex, TXID string

	transaction := Transaction{}
	transaction.ObjectType = TransactionObjectType
	transaction.TXStatus = "Canceled"
	transaction.TXMemo = "尚未完成比對"
	transaction.TXErrMsg = ""
	transaction.TXHcode = ""
	transaction.IsFrozen = false
	transaction.CreateTime = TimeNow
	transaction.UpdateTime = TimeNow

	err = checkArgArrayLength(args, 7)
	if err != nil {
		return transaction, false, "The args-length must be 7."
	}
	if len(args[0]) <= 0 {
		return transaction, false, "TXType must be a non-empty string."
	}
	if len(args[1]) <= 0 {
		return transaction, false, "TXFrom must be a non-empty string."
	}
	if len(args[2]) <= 0 {
		return transaction, false, "TXTo must be a non-empty string."
	}
	if len(args[3]) <= 0 {
		return transaction, false, "SecurityID must be a non-empty string."
	}
	if len(args[4]) <= 0 {
		return transaction, false, "SecurityAmount must be a non-empty string."
	}
	if len(args[5]) <= 0 {
		return transaction, false, "Payment must be a non-empty string."
	}
	if len(args[6]) <= 0 {
		return transaction, false, "isPutToQueue flag must be a non-empty string."
	}
	isPutToQueue, err := strconv.ParseBool(strings.ToLower(args[6]))
	if err != nil {
		return transaction, false, "isPutToQueue must be a boolean string."
	}
	transaction.isPutToQueue = isPutToQueue
	TXType := SubString(strings.ToUpper(args[0]), 0, 1)
	if (TXType != "B") && (TXType != "S") {
		return transaction, false, "TXType must be a B or S."
	}
	transaction.TXType = TXType
	TXFrom := strings.ToUpper(args[1])
	BankFrom := "BK" + SubString(TXFrom, 0, 3)
	TXID = BankFrom + TXType + TXFrom + TimeNow
	transaction.TXID = TXID
	TXTo := strings.ToUpper(args[2])
	BankTo := "BK" + SubString(TXTo, 0, 3)
	transaction.TXFrom = TXFrom
	transaction.TXTo = TXTo
	if TXFrom == TXTo {
		return transaction, false, "TXFrom can not equal to TXTo."
	}
	transaction.BankFrom = BankFrom
	transaction.BankTo = BankTo
	SecurityID := strings.ToUpper(args[3])
	_, err = getSecurityStructFromID(stub, SecurityID)

	if err != nil {
		return transaction, false, "SecurityID does not exits."
	}
	transaction.SecurityID = SecurityID
	SecurityAmount, err := strconv.ParseInt(args[4], 10, 64)
	if err != nil {
		return transaction, false, "SecurityAmount must be a numeric string."
	} else if SecurityAmount < 0 {
		return transaction, false, "SecurityAmount must be a positive value"
	}
	transaction.SecurityAmount = SecurityAmount
	Payment, err := strconv.ParseInt(args[5], 10, 64)
	if err != nil {
		return transaction, false, "Payment must be a numeric string"
	} else if Payment < 0 {
		return transaction, false, "Payment must be a positive value"
	}
	transaction.Payment = Payment
	if TXType == "S" {
		TXData1 = BankFrom + TXFrom + BankTo + TXTo + SecurityID + strconv.FormatInt(SecurityAmount, 10) + strconv.FormatInt(Payment, 10)
		TXIndex = getSHA256(TXData1)
		TXData2 = BankFrom + TXFrom + BankTo + TXTo + SecurityID
		TXSIndex = getSHA256(TXData2)
	}
	if TXType == "B" {
		TXData1 = BankTo + TXTo + BankFrom + TXFrom + SecurityID + strconv.FormatInt(SecurityAmount, 10) + strconv.FormatInt(Payment, 10)
		TXIndex = getSHA256(TXData1)
		TXData2 = BankTo + TXTo + BankFrom + TXFrom + SecurityID
		TXSIndex = getSHA256(TXData2)
	}
	transaction.TXIndex = TXIndex
	transaction.TXSIndex = TXSIndex
	balance, position, err := checkAccountBalance(stub, SecurityID, Payment, TXFrom, TXType)
	if err != nil {
		transaction.TXMemo = "券不足"
		transaction.TXErrMsg = TXFrom + ":Payment > Balance"
		fmt.Printf("Payment: %s\n", Payment)
		fmt.Printf("balance: %s\n", balance)
		fmt.Printf("position: %s\n", position)
		return transaction, false, TXFrom + ":Payment > Balance"
	}
	transaction.TXFromBalance = balance
	transaction.TXFromPosition = position

	transaction.TXStatus = "Pending"
	return transaction, isPutToQueue, ""

}

func getTransactionStructFromID(
	stub shim.ChaincodeStubInterface,
	TXID string) (*Transaction, error) {

	var errMsg string
	newTX := &Transaction{}
	TXAsBytes, err := stub.GetState(TXID)
	if err != nil {
		return newTX, err
	} else if TXAsBytes == nil {
		errMsg = fmt.Sprintf("Error: Transaction ID does not exist: %s", TXID)
		return newTX, errors.New(errMsg)
	}
	err = json.Unmarshal([]byte(TXAsBytes), newTX)
	if err != nil {
		return newTX, err
	}
	return newTX, nil
}

func getQueueStructFromID(
	stub shim.ChaincodeStubInterface,
	TXKEY string) (*QueuedTransaction, error) {

	var errMsg string
	queue := &QueuedTransaction{}
	queueAsBytes, err := stub.GetState(TXKEY)
	if err != nil {
		return nil, err
	} else if queueAsBytes == nil {
		errMsg = fmt.Sprintf("Error: QueuedTransaction ID does not exist: %s", TXKEY)
		return nil, errors.New(errMsg)
	}
	err = json.Unmarshal([]byte(queueAsBytes), queue)
	if err != nil {
		return nil, err
	}
	return queue, nil
}

func getHistoryTransactionStructFromID(
	stub shim.ChaincodeStubInterface,
	TXKEY string) (*TransactionHistory, error) {

	var errMsg string
	newTX := &TransactionHistory{}
	TXAsBytes, err := stub.GetState(TXKEY)
	if err != nil {
		return newTX, err
	} else if TXAsBytes == nil {
		errMsg = fmt.Sprintf("Error: TXKEY does not exist: %s", TXKEY)
		return newTX, errors.New(errMsg)
	}
	err = json.Unmarshal([]byte(TXAsBytes), newTX)
	if err != nil {
		return newTX, err
	}
	return newTX, nil
}

func getQueueArrayFromQuery(
	stub shim.ChaincodeStubInterface) ([]QueuedTransaction, int, error) {
	TimeNow := time.Now().Format(timelayout)
	//startKey := "20180000" //20180326
	//endKey := "20181231"
	var doflg bool
	var sumLen int
	doflg = false
	sumLen = 0
	TXKEY := SubString(TimeNow, 0, 8)
	resultsIterator, err := stub.GetStateByRange(TXKEY, TXKEY)
	if err != nil {
		return nil, sumLen, err
	}
	defer resultsIterator.Close()

	queueArr := []QueuedTransaction{}
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, sumLen, err
		}
		jsonByteObj := queryResponse.Value
		queue := QueuedTransaction{}
		for key, val := range queue.Transactions {
			if val.TXStatus != "" {
				json.Unmarshal(jsonByteObj, &queue)
				queueArr = append(queueArr, queue)
				doflg = true
				sumLen = key

			}
		}
	}
	if doflg != true {
		return nil, sumLen, err
	}
	return queueArr, sumLen, nil
}

func getSHA256(myData string) string {
	// ID generation
	moveOutInFundID := sha256.New()
	moveOutInFundID.Write([]byte(myData))
	moveOutInFundIDString := fmt.Sprintf("%x", moveOutInFundID.Sum(nil))
	return moveOutInFundIDString
}

func getMD5Str(myData string) string {
	h := md5.New()
	h.Write([]byte(myData))
	cipherStr := h.Sum(nil)
	encodeStr := hex.EncodeToString(cipherStr)
	return encodeStr
}

func checkAccountBalance(stub shim.ChaincodeStubInterface, SecurityID string, Payment int64, sender string, TXType string) (int64, int64, error) {
	senderAccount, err := getAccountStructFromID(stub, sender)
	var Balance int64
	var Position int64
	Balance = 0
	Position = 0
	if err != nil {
		return Balance, Position, err
	}

	var doflg bool
	doflg = false
	for key, val := range senderAccount.Assets {
		if val.SecurityID == SecurityID {
			Balance = senderAccount.Assets[key].Balance
			Position = senderAccount.Assets[key].Position
			if TXType == "S" {
				if (Payment > Balance) || (Payment > Position) {
					errMsg := fmt.Sprintf(
						"Error: Payment: (%s)  > Balance: (%s)",
						strconv.FormatInt(Payment, 10),
						strconv.FormatInt(Balance, 10))
					return Balance, Position, errors.New(errMsg)
				}
			}
			doflg = true
			break
		}
	}
	if doflg != true {
		errMsg := fmt.Sprintf(
			"Error: This SecurityID does not exists (%s)",
			SecurityID)
		return Balance, Position, errors.New(errMsg)
	}

	return Balance, Position, nil
}

func updateAccountBalance(stub shim.ChaincodeStubInterface, SecurityID string, SecurityAmount int64, Payment int64, sender string, receiver string) (int64, int64, error) {
	senderAccount, err := getAccountStructFromID(stub, sender)
	receiverAccount, err := getAccountStructFromID(stub, receiver)
	var senderBalance int64
	var receiverBalance int64
	if err != nil {
		return senderBalance, receiverBalance, err
	}

	var doflg bool
	doflg = false
	for key, val := range senderAccount.Assets {
		if val.SecurityID == SecurityID {
			senderAccount.Assets[key].SecurityAmount += SecurityAmount
			senderAccount.Assets[key].Balance -= Payment
			senderAccount.Assets[key].Position -= Payment
			senderBalance = senderAccount.Assets[key].Balance
			doflg = true
			break
		}
	}
	if doflg != true {
		errMsg := fmt.Sprintf(
			"Error: This SecurityID does not exists (%s)",
			SecurityID)
		return senderBalance, receiverBalance, errors.New(errMsg)
	}
	if doflg == true {
		for key, val := range receiverAccount.Assets {
			if val.SecurityID == SecurityID {
				receiverAccount.Assets[key].SecurityAmount -= SecurityAmount
				receiverAccount.Assets[key].Balance += Payment
				receiverAccount.Assets[key].Position += Payment
				receiverBalance = receiverAccount.Assets[key].Balance
				doflg = true
				break
			}
		}
	}

	if senderBalance >= 0 {
		senderAccountAsBytes, err := json.Marshal(senderAccount)
		err = stub.PutState(sender, senderAccountAsBytes)
		if err != nil {
			return senderBalance, receiverBalance, err
		}
	}
	if receiverBalance >= 0 {
		receiverAccountAsBytes, err := json.Marshal(receiverAccount)
		err = stub.PutState(receiver, receiverAccountAsBytes)
		if err != nil {
			return senderBalance, receiverBalance, err
		}
	}
	fmt.Printf("5.senderBalance= %d\n", senderBalance)
	fmt.Printf("6.receiverBalance= %d\n", receiverBalance)

	return senderBalance, receiverBalance, nil
}

func updateSecurityAmount(stub shim.ChaincodeStubInterface, SecurityID string, Payment int64, sender string, receiver string) (int64, int64, error) {
	Security, err := getSecurityStructFromID(stub, SecurityID)
	fmt.Printf("updateSecurityAmount, SecurityID=%s,Payment=%d,sender=%s,receiver=%s\n", SecurityID, Payment, sender, receiver)

	var senderBalance int64
	var receiverBalance int64
	if err != nil {
		return senderBalance, receiverBalance, err
	}
	senderBank := SubString(sender, 0, 3)
	receiverBank := SubString(receiver, 0, 3)

	var doflg bool
	doflg = false
	for key, val := range Security.Owners {
		if val.OwnedAccountID == sender {
			Security.Owners[key].OwnedAmount -= Payment
			senderBalance = Security.Owners[key].OwnedAmount
			doflg = true
		}
		if val.OwnedAccountID == receiver {
			Security.Owners[key].OwnedAmount += Payment
			receiverBalance = Security.Owners[key].OwnedAmount
			doflg = true
		}
	}
	if doflg != true {
		errMsg := fmt.Sprintf(
			"Error: This OwnedAccountID does not exists (%s)",
			sender)
		return senderBalance, receiverBalance, errors.New(errMsg)
	}
	doflg = false
	TimeNow := time.Now().Format(timelayout)

	for key, val := range Security.SecurityTotals {
		fmt.Printf("1.Skey: %d\n", key)
		fmt.Printf("2.Sval: %s\n", val)
		if val.BankID == senderBank {
			fmt.Printf("3.Skey: %d\n", key)
			fmt.Printf("4.Sval: %s\n", val)
			Security.SecurityTotals[key].TotalBalance -= Payment
			Security.SecurityTotals[key].UpdateTime = TimeNow
			doflg = true
		}
		if val.BankID == receiverBank {
			fmt.Printf("5.Skey: %d\n", key)
			fmt.Printf("6.Sval: %s\n", val)
			Security.SecurityTotals[key].TotalBalance += Payment
			Security.SecurityTotals[key].UpdateTime = TimeNow
			doflg = true
		}
	}

	if senderBalance >= 0 || receiverBalance >= 0 {
		SecurityAsBytes, err := json.Marshal(Security)
		err = stub.PutState(SecurityID, SecurityAsBytes)
		if err != nil {
			return senderBalance, receiverBalance, err
		}
	}

	fmt.Printf("7.senderBalance= %d\n", senderBalance)
	fmt.Printf("8.receiverBalance= %d\n", receiverBalance)
	fmt.Printf("9.SecurityID= %s\n", SecurityID)

	return senderBalance, receiverBalance, nil
}

func resetSecurityAmount(stub shim.ChaincodeStubInterface, SecurityID string, Payment int64, sender string, receiver string) (int64, int64, error) {

	fmt.Printf("resetSecurityAmount, SecurityID=%s,Payment=%d,sender=%s,receiver=%s\n", SecurityID, Payment, sender, receiver)
	Security, err := getSecurityStructFromID(stub, SecurityID)

	var senderBalance int64
	var receiverBalance int64
	if err != nil {
		return senderBalance, receiverBalance, err
	}
	senderBank := SubString(sender, 0, 3)
	receiverBank := SubString(receiver, 0, 3)

	var doflg bool
	doflg = false
	for key, val := range Security.Owners {
		if val.OwnedAccountID == sender {
			Security.Owners[key].OwnedAmount += Payment
			senderBalance = Security.Owners[key].OwnedAmount
			doflg = true
		}
		if val.OwnedAccountID == receiver {
			Security.Owners[key].OwnedAmount -= Payment
			receiverBalance = Security.Owners[key].OwnedAmount
			doflg = true
		}
	}
	if doflg != true {
		errMsg := fmt.Sprintf(
			"Error: This OwnedAccountID does not exists (%s)",
			sender)
		return senderBalance, receiverBalance, errors.New(errMsg)
	}
	doflg = false
	TimeNow := time.Now().Format(timelayout)

	for key, val := range Security.SecurityTotals {
		fmt.Printf("1.Skey: %d\n", key)
		fmt.Printf("2.Sval: %s\n", val)
		if val.BankID == senderBank {
			fmt.Printf("3.Skey: %d\n", key)
			fmt.Printf("4.Sval: %s\n", val)
			Security.SecurityTotals[key].TotalBalance += Payment
			Security.SecurityTotals[key].UpdateTime = TimeNow
			doflg = true
		}
		if val.BankID == receiverBank {
			fmt.Printf("5.Skey: %d\n", key)
			fmt.Printf("6.Sval: %s\n", val)
			Security.SecurityTotals[key].TotalBalance -= Payment
			Security.SecurityTotals[key].UpdateTime = TimeNow
			doflg = true
		}
	}

	if senderBalance >= 0 || receiverBalance >= 0 {
		SecurityAsBytes, err := json.Marshal(Security)
		err = stub.PutState(SecurityID, SecurityAsBytes)
		if err != nil {
			return senderBalance, receiverBalance, err
		}
	}

	fmt.Printf("7.senderBalance= %d\n", senderBalance)
	fmt.Printf("8.receiverBalance= %d\n", receiverBalance)
	fmt.Printf("9.SecurityID= %s\n", SecurityID)

	return senderBalance, receiverBalance, nil
}

func updateTransactionStatus(stub shim.ChaincodeStubInterface, TXID string, TXStatus string, MatchedTXID string) error {
	fmt.Printf("1 updateTransactionStatus TXID = %s, TXStatus = %s, MatchedTXID = %s\n", TXID, TXStatus, MatchedTXID)
	TimeNow := time.Now().Format(timelayout)
	transaction, err := getTransactionStructFromID(stub, TXID)
	transaction.TXStatus = TXStatus

	var TXMemo, TXErrMsg string
	TXMemo = ""
	TXErrMsg = ""
	if TXStatus == "PaymentError" {
		TXMemo = "同資款不足"
		transaction.TXMemo = TXMemo
	}
	if TXStatus == "Cancelled" {
		TXMemo = "取消交易"
		transaction.TXMemo = TXMemo
	}
	if TXStatus == "Matched" {
		TXMemo = ""
		transaction.TXMemo = TXMemo
	}
	if TXStatus == "Finished" {
		TXMemo = ""
		transaction.TXMemo = TXMemo
		transaction.TXErrMsg = TXErrMsg
	}

	if TXStatus != "Cancelled" && TXStatus != "PaymentError" {
		transaction.IsFrozen = true
	} else {
		transaction.IsFrozen = false
	}
	fmt.Printf("3 updateTransactionStatus MatchedTXID = %s\n", MatchedTXID)
	transaction.MatchedTXID = MatchedTXID
	fmt.Printf("4 updateTransactionStatus transaction MatchedTXID = %s\n", transaction.MatchedTXID)

	transaction.UpdateTime = TimeNow
	transactionAsBytes, err := json.Marshal(transaction)
	if err != nil {
		return err
	}
	err = stub.PutState(TXID, transactionAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateQueuedTransactionStatus(stub shim.ChaincodeStubInterface, TXKEY string, TXID string, TXStatus string) error {

	TimeNow := time.Now().Format(timelayout)
	queuedTX, err := getQueueStructFromID(stub, TXKEY)
	if err != nil {
		return err
	}
	var doflg bool
	doflg = false

	for key, val := range queuedTX.TXIDs {
		if val == TXID {
			queuedTX.Transactions[key].TXStatus = TXStatus
			queuedTX.Transactions[key].UpdateTime = TimeNow
			doflg = true
			break
		}
	}
	if doflg != true {
		return errors.New("Failed to find Queued TXID ")
	}

	queuedAsBytes, err := json.Marshal(queuedTX)
	if err != nil {
		return err
	}
	err = stub.PutState(TXKEY, queuedAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateHistoryTransactionStatus(stub shim.ChaincodeStubInterface, HTXKEY string, TXID string, TXStatus string) error {

	TimeNow := time.Now().Format(timelayout)
	historyTX, err := getHistoryTransactionStructFromID(stub, HTXKEY)
	if err != nil {
		return err
	}
	var doflg bool
	doflg = false

	for key, val := range historyTX.TXIDs {
		if val == TXID {
			historyTX.Transactions[key].TXStatus = TXStatus
			historyTX.Transactions[key].UpdateTime = TimeNow
			historyTX.TXStatus[key] = TXStatus
			doflg = true
			break
		}
	}
	if doflg != true {
		return errors.New("Failed to find History TXID ")
	}

	historyAsBytes, err := json.Marshal(historyTX)
	if err != nil {
		return err
	}
	err = stub.PutState(HTXKEY, historyAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateQueuedTransactionApproveStatus(stub shim.ChaincodeStubInterface, TXKEY string, TXID string, MatchedTXID string, TXStatus string) error {

	TimeNow := time.Now().Format(timelayout)
	queuedTX, err := getQueueStructFromID(stub, TXKEY)
	if err != nil {
		return err
	}
	var doflg1, doflg2 bool
	doflg1 = false
	doflg2 = false
	var TXMemo, TXErrMsg string
	TXMemo = ""
	TXErrMsg = ""
	if TXStatus == "PaymentError" {
		TXMemo = "同資款不足"
	}
	if TXStatus == "Cancelled" {
		TXMemo = "取消交易"
	}
	if TXStatus == "Matched" {
		TXMemo = ""
	}
	if TXStatus == "Finished" {
		TXMemo = ""
	}

	for key, val := range queuedTX.TXIDs {
		fmt.Printf("1.ATXIDs: %s\n", TXID)
		fmt.Printf("2.AMatchedTXID: %s\n", MatchedTXID)
		if TXStatus == "Finished" {
			queuedTX.Transactions[key].TXErrMsg = TXErrMsg
		}
		if val == TXID {
			fmt.Printf("3.AQkey: %d\n", key)
			fmt.Printf("4.AQval: %s\n", val)
			queuedTX.Transactions[key].TXStatus = TXStatus
			queuedTX.Transactions[key].TXMemo = TXMemo
			queuedTX.Transactions[key].UpdateTime = TimeNow
			doflg1 = true
		}
		if val == MatchedTXID {
			fmt.Printf("5.AQkey: %d\n", key)
			fmt.Printf("6.AQval: %s\n", val)
			queuedTX.Transactions[key].TXStatus = TXStatus
			queuedTX.Transactions[key].TXMemo = TXMemo
			queuedTX.Transactions[key].UpdateTime = TimeNow
			doflg2 = true
		}
		if doflg1 == true && doflg2 == true {
			break
		}
	}
	if doflg1 != true || doflg2 != true {
		return errors.New("Failed to find Approve-Queued TXID ")
	}

	queuedAsBytes, err := json.Marshal(queuedTX)
	if err != nil {
		return err
	}
	err = stub.PutState(TXKEY, queuedAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateHistoryTransactionApproveStatus(stub shim.ChaincodeStubInterface, HTXKEY string, TXID string, MatchedTXID string, TXStatus string) error {

	TimeNow := time.Now().Format(timelayout)
	historyTX, err := getHistoryTransactionStructFromID(stub, HTXKEY)
	if err != nil {
		return err
	}
	var doflg1, doflg2 bool
	doflg1 = false
	doflg2 = false
	var TXMemo, TXErrMsg string
	TXMemo = ""
	TXErrMsg = ""
	if TXStatus == "PaymentError" {
		TXMemo = "同資款不足"
	}
	if TXStatus == "Cancelled" {
		TXMemo = "取消交易"
	}
	if TXStatus == "Matched" {
		TXMemo = ""
	}
	if TXStatus == "Finished" {
		TXMemo = ""
	}

	for key, val := range historyTX.TXIDs {
		fmt.Printf("7.ATXIDs: %s\n", TXID)
		fmt.Printf("8.AMatchedTXID: %s\n", MatchedTXID)
		if TXStatus == "Finished" {
			historyTX.Transactions[key].TXErrMsg = TXErrMsg
		}
		if val == TXID {
			fmt.Printf("9.AHkey: %d\n", key)
			fmt.Printf("10.AHval: %s\n", val)
			historyTX.Transactions[key].TXStatus = TXStatus
			historyTX.Transactions[key].TXMemo = TXMemo
			historyTX.Transactions[key].UpdateTime = TimeNow
			historyTX.TXStatus[key] = TXStatus
			doflg1 = true
		}
		if val == MatchedTXID {
			fmt.Printf("11.AHkey: %d\n", key)
			fmt.Printf("12.AHval: %s\n", val)
			historyTX.Transactions[key].TXStatus = TXStatus
			historyTX.Transactions[key].TXMemo = TXMemo
			historyTX.Transactions[key].UpdateTime = TimeNow
			historyTX.TXStatus[key] = TXStatus
			doflg2 = true
		}
		if doflg1 == true && doflg2 == true {
			break
		}
	}
	if doflg1 != true || doflg2 != true {
		return errors.New("Failed to find Approve-History TXID ")
	}

	historyAsBytes, err := json.Marshal(historyTX)
	if err != nil {
		return err
	}
	err = stub.PutState(HTXKEY, historyAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func resetAccountBalance(stub shim.ChaincodeStubInterface, SecurityID string, SecurityAmount int64, Payment int64, sender string, receiver string) (int64, int64, error) {
	senderAccount, err := getAccountStructFromID(stub, sender)
	receiverAccount, err := getAccountStructFromID(stub, receiver)
	var senderBalance int64
	var receiverBalance int64
	if err != nil {
		return senderBalance, receiverBalance, err
	}

	var doflg bool
	doflg = false
	for key, val := range senderAccount.Assets {
		if val.SecurityID == SecurityID {
			senderAccount.Assets[key].SecurityAmount -= SecurityAmount
			senderAccount.Assets[key].Balance += Payment
			senderAccount.Assets[key].Position += Payment
			senderBalance = senderAccount.Assets[key].Balance
			doflg = true
			break
		}
	}
	if doflg != true {
		errMsg := fmt.Sprintf(
			"Error: This SecurityID does not exists (%s)",
			SecurityID)
		return senderBalance, receiverBalance, errors.New(errMsg)
	}
	if doflg == true {
		for key, val := range receiverAccount.Assets {
			if val.SecurityID == SecurityID {
				receiverAccount.Assets[key].SecurityAmount += SecurityAmount
				receiverAccount.Assets[key].Balance -= Payment
				receiverAccount.Assets[key].Position -= Payment
				receiverBalance = receiverAccount.Assets[key].Balance
				doflg = true
				break
			}
		}
	}

	if senderBalance >= 0 {
		senderAccountAsBytes, err := json.Marshal(senderAccount)
		err = stub.PutState(sender, senderAccountAsBytes)
		if err != nil {
			return senderBalance, receiverBalance, err
		}
	}
	if receiverBalance >= 0 {
		receiverAccountAsBytes, err := json.Marshal(receiverAccount)
		err = stub.PutState(receiver, receiverAccountAsBytes)
		if err != nil {
			return senderBalance, receiverBalance, err
		}
	}

	return senderBalance, receiverBalance, nil
}

func updateTransactionTXHcode(stub shim.ChaincodeStubInterface, TXID string, TXHcode string) error {
	fmt.Printf("updateTransactionTXHcode: TXID=%s,TXHcode=%s\n", TXID, TXHcode)

	TimeNow := time.Now().Format(timelayout)
	transaction, err := getTransactionStructFromID(stub, TXID)
	transaction.TXHcode = TXHcode
	transaction.TXStatus = "Cancelled"
	transaction.TXMemo = "更正交易"
	transaction.UpdateTime = TimeNow

	transactionAsBytes, err := json.Marshal(transaction)
	if err != nil {
		return err
	}
	err = stub.PutState(TXID, transactionAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateQueuedTransactionTXHcode(stub shim.ChaincodeStubInterface, TXKEY string, TXID string, TXHcode string) error {
	fmt.Printf("updateQueuedTransactionTXHcode: TXKEY=%s,TXID=%s,TXHcode=%s\n", TXKEY, TXID, TXHcode)
	TimeNow := time.Now().Format(timelayout)
	queuedTX, err := getQueueStructFromID(stub, TXKEY)
	if err != nil {
		return err
	}
	var doflg bool
	doflg = false

	for key, val := range queuedTX.TXIDs {
		fmt.Printf("1.Qkey: %d\n", key)
		fmt.Printf("2.Qval: %s\n", val)
		if val == TXID {
			fmt.Printf("3.Qkey: %d\n", key)
			fmt.Printf("4.Qval: %s\n", val)
			queuedTX.Transactions[key].TXHcode = TXHcode
			queuedTX.Transactions[key].TXStatus = "Cancelled"
			queuedTX.Transactions[key].TXMemo = "更正交易"
			queuedTX.Transactions[key].UpdateTime = TimeNow
			doflg = true
			break
		}
	}
	if doflg != true {
		return errors.New("5.Failed to find Queued TXID ")
	}

	queuedAsBytes, err := json.Marshal(queuedTX)
	if err != nil {
		return err
	}
	err = stub.PutState(TXKEY, queuedAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateHistoryTransactionTXHcode(stub shim.ChaincodeStubInterface, HTXKEY string, TXID string, TXHcode string) error {
	fmt.Printf("updateHistoryTransactionTXHcode: HTXKEY=%s,TXID=%s,TXHcode=%s\n", HTXKEY, TXID, TXHcode)
	TimeNow := time.Now().Format(timelayout)
	historyTX, err := getHistoryTransactionStructFromID(stub, HTXKEY)
	if err != nil {
		return err
	}
	//TimeNow := time.Now().Format(timelayout)

	var doflg bool
	doflg = false

	for key, val := range historyTX.TXIDs {
		fmt.Printf("1.Hkey: %d\n", key)
		fmt.Printf("2.Hval: %s\n", val)
		if val == TXID {
			fmt.Printf("3.Hkey: %d\n", key)
			fmt.Printf("4.Hval: %s\n", val)
			historyTX.Transactions[key].TXHcode = TXHcode
			historyTX.Transactions[key].TXStatus = "Cancelled"
			historyTX.Transactions[key].TXMemo = "更正交易"
			historyTX.TXStatus[key] = "Cancelled"
			historyTX.Transactions[key].UpdateTime = TimeNow
			doflg = true
			break
		}
	}
	if doflg != true {
		return errors.New("Failed to find History TXID ")
	}

	historyAsBytes, err := json.Marshal(historyTX)
	if err != nil {
		return err
	}
	err = stub.PutState(HTXKEY, historyAsBytes)
	if err != nil {
		return err
	}
	return nil
}

//peer chaincode invoke -n mycc1 -c '{"Args":["securityCorrectTransfer", "S","004000000001" , "002000000001" , "A07106" , "102000","100000","true","BANK002B00200000000120180606155851"]}' -C myc
func (s *SmartContract) securityCorrectTransfer(
	stub shim.ChaincodeStubInterface,
	args []string) peer.Response {

	TimeNow := time.Now().Format(timelayout)
	newTX, isPutInQueue, errMsg := validateCorrectTransaction(stub, args)
	if errMsg != "" {
		//return shim.Error(err.Error())
		newTX.TXErrMsg = errMsg
	}
	TXIndex := newTX.TXIndex
	TXID := newTX.TXID
	TXType := newTX.TXType
	SecurityID := newTX.SecurityID
	TXFrom := newTX.TXFrom
	TXTo := newTX.TXTo
	BankFrom := newTX.BankFrom
	BankTo := newTX.BankTo
	Payment := newTX.Payment
	SecurityAmount := newTX.SecurityAmount
	TXStatus := newTX.TXStatus
	TXHcode := newTX.TXHcode

	var doflg bool
	var TXKinds string
	doflg = false
	TXKEY := SubString(TimeNow, 0, 8) //A0710220180326
	HTXKEY := "H" + TXKEY
	if BankFrom != BankTo {
		if SecurityAmount == 0 {
			if TXType == "S" {
				TXKinds = "跨行FOP轉出"
			} else {
				TXKinds = "跨行FOP轉入"
			}
		} else {
			if TXType == "S" {
				TXKinds = "跨行DVP轉出"
			} else {
				TXKinds = "跨行DVP轉入"
			}
		}
	} else {
		if TXType == "S" {
			TXKinds = "自行FOP轉出"
		} else {
			TXKinds = "自行FOP轉入"
		}
	}
	ApproveFlag := approved0
	ValueAsBytes, err := stub.GetState("approveflag")
	if err == nil {
		ApproveFlag = string(ValueAsBytes)
	}
	fmt.Printf("1.ApproveFlag=%s\n", ApproveFlag)

	if isPutInQueue == true {

		queueAsBytes, err := stub.GetState(TXKEY)
		if err != nil {
			//return shim.Error(err.Error())
			newTX.TXErrMsg = TXKEY + ":QueueID does not exits."
		}
		queuedTx := QueuedTransaction{}
		json.Unmarshal(queueAsBytes, &queuedTx)

		historyAsBytes, err := stub.GetState(HTXKEY)
		if err != nil {
			//return shim.Error(err.Error())
			newTX.TXErrMsg = TXKEY + ":HistoryID does not exits."
		}
		historyNewTX := TransactionHistory{}
		json.Unmarshal(historyAsBytes, &historyNewTX)

		fmt.Println("01.CTXIndex= " + TXIndex + "\n")
		fmt.Println("02.CTXFrom= " + TXFrom + "\n")
		fmt.Println("03.CTXType= " + TXType + "\n")
		fmt.Println("04.CTXID= " + TXID + "\n")
		fmt.Println("05.CTXStatus= " + TXStatus + "\n")
		fmt.Println("06.Cval.TXHcode= " + TXHcode + "\n")

		if queueAsBytes == nil {
			queuedTx.ObjectType = QueuedTXObjectType
			queuedTx.TXKEY = TXKEY
			queuedTx.Transactions = append(queuedTx.Transactions, newTX)
			queuedTx.TXIndexs = append(queuedTx.TXIndexs, TXIndex)
			queuedTx.TXIDs = append(queuedTx.TXIDs, TXID)
			if historyAsBytes == nil {
				historyNewTX.ObjectType = HistoryTXObjectType
				historyNewTX.TXKEY = HTXKEY
				historyNewTX.Transactions = append(historyNewTX.Transactions, newTX)
				historyNewTX.TXIndexs = append(historyNewTX.TXIndexs, TXIndex)
				historyNewTX.TXIDs = append(historyNewTX.TXIDs, TXID)
				historyNewTX.TXStatus = append(historyNewTX.TXStatus, newTX.TXStatus)
				historyNewTX.TXKinds = append(historyNewTX.TXKinds, TXKinds)
			}
		} else if queueAsBytes != nil {
			for key, val := range queuedTx.TXIDs {
				fmt.Printf("qkey1: %d\n", key)
				fmt.Printf("qval1: %s\n", val)

				if val == TXHcode {

					if queuedTx.Transactions[key].TXStatus == "Pending" {
						fmt.Printf("qkey2: %d\n", key)
						fmt.Printf("qval2: %s\n", val)
						queuedTx.Transactions[key].TXStatus = "Cancelled"
						queuedTx.Transactions[key].TXHcode = TXID
						queuedTx.Transactions[key].UpdateTime = TimeNow
						doflg = true
					}
					if historyNewTX.Transactions[key].TXStatus == "Pending" {
						fmt.Printf("hkey2: %d\n", key)
						fmt.Printf("hval2: %s\n", val)
						historyNewTX.TXStatus[key] = "Cancelled"
						historyNewTX.Transactions[key].TXStatus = "Cancelled"
						historyNewTX.Transactions[key].TXHcode = TXID
						historyNewTX.Transactions[key].UpdateTime = TimeNow
						doflg = true
					}
					if doflg == true {
						doflg = false
						break
					}
				}
			}

			for key, val := range queuedTx.Transactions {
				//if val.TXIndex == TXIndex && val.TXFrom != TXFrom && val.TXType != TXType {
				if val.TXIndex == TXIndex && val.TXStatus == TXStatus && val.TXFrom != TXFrom && val.TXType != TXType && val.TXID != TXID {
					fmt.Println("11.CTXIndex= " + TXIndex + "\n")
					fmt.Println("12.CTXFrom= " + TXFrom + "\n")
					fmt.Println("13.CTXType= " + TXType + "\n")
					fmt.Println("14.CTXID= " + TXID + "\n")
					fmt.Println("15.Cval.TXID= " + val.TXID + "\n")
					fmt.Println("16.CTXStatus= " + TXStatus + "\n")
					fmt.Println("17.Cval.TXStatus= " + val.TXStatus + "\n")
					fmt.Println("18.Cval.TXHcode= " + TXHcode + "\n")

					if TXStatus == "Pending" && val.TXStatus == "Pending" { //if queuedTx.Transactions[key].TXStatus == "Pending" && TXStatus == "Pending" {
						if doflg == true {
							//return shim.Error("doflg eq to true")
							newTX.TXErrMsg = "doflg eq to true"
						}
						newTX.MatchedTXID = val.TXID
						queuedTx.Transactions[key].MatchedTXID = TXID
						historyNewTX.Transactions[key].MatchedTXID = TXID
						err = updateTransactionStatus(stub, val.TXID, "Matched", TXID)
						if err != nil {
							//return shim.Error(err.Error())
							newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Matched."
						}
						queuedTx.Transactions[key].TXStatus = "Matched"
						historyNewTX.Transactions[key].TXStatus = "Matched"
						historyNewTX.TXStatus[key] = "Matched"
						newTX.TXStatus = "Matched"
						queuedTx.Transactions[key].TXMemo = ""
						historyNewTX.Transactions[key].TXMemo = ""
						newTX.TXMemo = ""
						if TXType == "S" {
							//轉出          轉入
							senderBalance, receiverBalance, err := updateAccountBalance(stub, SecurityID, SecurityAmount, Payment, TXFrom, TXTo)
							senderBalance, receiverBalance, err = updateSecurityAmount(stub, SecurityID, Payment, TXFrom, TXTo)
							err = updateBankTotals(stub, TXFrom, SecurityID, Payment, true)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateBankTotals TXFrom:" + TXFrom
							}
							err = updateBankTotals(stub, TXTo, SecurityID, Payment, false)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateBankTotals TXTo:" + TXTo
							}
							if (senderBalance < 0) || (receiverBalance < 0) {
								//return shim.Error("TxType=S - senderBalance or receiverBalance <0")
								newTX.TXErrMsg = "TxType=S - senderBalance or receiverBalance <0"
							}
						}
						if TXType == "B" {
							//轉出          轉入
							senderBalance, receiverBalance, err := updateAccountBalance(stub, SecurityID, SecurityAmount, Payment, TXTo, TXFrom)
							senderBalance, receiverBalance, err = updateSecurityAmount(stub, SecurityID, Payment, TXTo, TXFrom)
							err = updateBankTotals(stub, TXFrom, SecurityID, Payment, true)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateBankTotals TXFrom:" + TXFrom
							}
							err = updateBankTotals(stub, TXTo, SecurityID, Payment, false)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateBankTotals TXTo:" + TXTo
							}
							if (senderBalance < 0) || (receiverBalance < 0) {
								//return shim.Error("TxType=B - senderBalance or receiverBalance <0")
								newTX.TXErrMsg = "TxType=B - senderBalance or receiverBalance <0"
							}
						}
						newTX.IsFrozen = true
						queuedTx.Transactions[key].IsFrozen = true
						historyNewTX.Transactions[key].IsFrozen = true
						if BankFrom != BankTo {
							if SecurityAmount != 0 {
								if ApproveFlag == approved0 {
									queuedTx.Transactions[key].TXStatus = "Finished"
									historyNewTX.Transactions[key].TXStatus = "Finished"
									historyNewTX.TXStatus[key] = "Finished"
									newTX.TXStatus = "Finished"
									err := updateTransactionStatus(stub, val.TXID, "Finished", TXID)
									if err != nil {
										//return shim.Error(err.Error())
										newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Finished."
									}
								} else {
									queuedTx.Transactions[key].TXStatus = "Waiting4Payment"
									historyNewTX.Transactions[key].TXStatus = "Waiting4Payment"
									historyNewTX.TXStatus[key] = "Waiting4Payment"
									newTX.TXStatus = "Waiting4Payment"
									err := updateTransactionStatus(stub, val.TXID, "Waiting4Payment", TXID)
									if err != nil {
										//return shim.Error(err.Error())
										newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Waiting4Payment."
									}
								}
							} else {
								queuedTx.Transactions[key].TXStatus = "Finished"
								historyNewTX.Transactions[key].TXStatus = "Finished"
								historyNewTX.TXStatus[key] = "Finished"
								newTX.TXStatus = "Finished"
								err := updateTransactionStatus(stub, val.TXID, "Finished", TXID)
								if err != nil {
									//return shim.Error(err.Error())
									newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Finished."
								}
							}
						} else {
							queuedTx.Transactions[key].TXStatus = "Finished"
							historyNewTX.Transactions[key].TXStatus = "Finished"
							historyNewTX.TXStatus[key] = "Finished"
							newTX.TXStatus = "Finished"
							err := updateTransactionStatus(stub, val.TXID, "Finished", TXID)
							if err != nil {
								//return shim.Error(err.Error())
								newTX.TXErrMsg = "Failed to execute updateTransactionStatus = Finished."
							}
						}

						doflg = true
						break
					}
				}
			}
		}
		if queueAsBytes != nil {
			if historyAsBytes != nil {
				queuedTx.ObjectType = QueuedTXObjectType
				queuedTx.TXKEY = TXKEY
				queuedTx.Transactions = append(queuedTx.Transactions, newTX)
				queuedTx.TXIndexs = append(queuedTx.TXIndexs, TXIndex)
				queuedTx.TXIDs = append(queuedTx.TXIDs, TXID)

				historyNewTX.ObjectType = HistoryTXObjectType
				historyNewTX.TXKEY = HTXKEY
				historyNewTX.Transactions = append(historyNewTX.Transactions, newTX)
				historyNewTX.TXIndexs = append(historyNewTX.TXIndexs, TXIndex)
				historyNewTX.TXIDs = append(historyNewTX.TXIDs, TXID)
				historyNewTX.TXStatus = append(historyNewTX.TXStatus, newTX.TXStatus)
				historyNewTX.TXKinds = append(historyNewTX.TXKinds, TXKinds)
			}
		}
		QueuedAsBytes, err := json.Marshal(queuedTx)
		err = stub.PutState(TXKEY, QueuedAsBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
		historyAsBytes, err = json.Marshal(historyNewTX)
		err = stub.PutState(HTXKEY, historyAsBytes)
		if err != nil {
			return shim.Error(err.Error())
		}
	}
	TransactionAsBytes, err := json.Marshal(newTX)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.PutState(TXID, TransactionAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)

}

func validateCorrectTransaction(
	stub shim.ChaincodeStubInterface,
	args []string) (Transaction, bool, string) {
	TimeNow := time.Now().Format(timelayout)
	var err error
	var TXData1, TXData2, TXIndex, TXSIndex, TXID string
	//TimeNow := time.Now().Format(timelayout)
	transaction := Transaction{}
	transaction.ObjectType = TransactionObjectType
	transaction.TXStatus = "Canceled"
	transaction.TXMemo = "更正交易"
	transaction.TXErrMsg = ""
	transaction.TXHcode = ""
	transaction.IsFrozen = false
	transaction.CreateTime = TimeNow
	transaction.UpdateTime = TimeNow
	fmt.Println("TimeNow is %s", TimeNow)

	err = checkArgArrayLength(args, 8)
	if err != nil {
		return transaction, false, "The args-length must be 8."
	}
	if len(args[0]) <= 0 {
		return transaction, false, "TXType must be a non-empty string."
	}
	if len(args[1]) <= 0 {
		return transaction, false, "TXFrom must be a non-empty string."
	}
	if len(args[2]) <= 0 {
		return transaction, false, "TXTo must be a non-empty string."
	}
	if len(args[3]) <= 0 {
		return transaction, false, "SecurityID must be a non-empty string."
	}
	if len(args[4]) <= 0 {
		return transaction, false, "SecurityAmount must be a non-empty string."
	}
	if len(args[5]) <= 0 {
		return transaction, false, "Payment must be a non-empty string."
	}
	if len(args[6]) <= 0 {
		return transaction, false, "isPutToQueue flag must be a non-empty string."
	}
	if len(args[7]) <= 0 {
		return transaction, false, "TXID flag must be a non-empty string."
	}
	TXID = strings.ToUpper(args[7])
	sourceTX, err := getTransactionStructFromID(stub, TXID)
	if sourceTX.TXStatus != "Pending" {
		return transaction, false, "Failed to find Transaction Pending TXStatus."
	}

	TXType := args[0]
	if (TXType != "B") && (TXType != "S") {
		return transaction, false, "TXType must be a B or S."
	}
	transaction.TXType = TXType
	TXFrom := strings.ToUpper(args[1])
	BankFrom := "BK" + SubString(TXFrom, 0, 3)
	transaction.TXFrom = TXFrom
	transaction.BankFrom = BankFrom
	TXHcode := BankFrom + TXType + TXFrom + TimeNow
	transaction.TXID = TXHcode
	TXTo := strings.ToUpper(args[2])
	BankTo := "BK" + SubString(TXTo, 0, 3)
	transaction.TXTo = TXTo
	transaction.BankTo = BankTo
	if TXFrom == TXTo {
		return transaction, false, "TXFrom equal to TXTo."
	}

	SecurityID := strings.ToUpper(args[3])
	_, err = getSecurityStructFromID(stub, SecurityID)
	if err != nil {
		return transaction, false, "SecurityID does not exits."
	}
	transaction.SecurityID = SecurityID
	SecurityAmount, err := strconv.ParseInt(args[4], 10, 64)
	if err != nil {
		return transaction, false, "SecurityAmount must be a numeric string."
	} else if SecurityAmount < 0 {
		return transaction, false, "SecurityAmount must be a positive value."
	}
	transaction.SecurityAmount = SecurityAmount
	Payment, err := strconv.ParseInt(args[5], 10, 64)
	if err != nil {
		return transaction, false, "Payment must be a numeric string."
	} else if Payment < 0 {
		return transaction, false, "Payment must be a positive value."
	}
	transaction.Payment = Payment
	isPutToQueue, err := strconv.ParseBool(strings.ToLower(args[6]))
	if err != nil {
		return transaction, false, "isPutToQueue must be a boolean string."
	}
	transaction.isPutToQueue = isPutToQueue

	if TXType == "S" {
		TXData1 = BankFrom + TXFrom + BankTo + TXTo + SecurityID + strconv.FormatInt(SecurityAmount, 10) + strconv.FormatInt(Payment, 10)
		TXIndex = getSHA256(TXData1)
		TXData2 = BankFrom + TXFrom + BankTo + TXTo + SecurityID
		TXSIndex = getSHA256(TXData2)
	}

	if TXType == "B" {
		TXData1 = BankTo + TXTo + BankFrom + TXFrom + SecurityID + strconv.FormatInt(SecurityAmount, 10) + strconv.FormatInt(Payment, 10)
		TXIndex = getSHA256(TXData1)
		TXData2 = BankTo + TXTo + BankFrom + TXFrom + SecurityID
		TXSIndex = getSHA256(TXData2)
	}
	transaction.TXIndex = TXIndex
	transaction.TXSIndex = TXSIndex
	balance, position, err := checkAccountBalance(stub, SecurityID, Payment, TXFrom, TXType)
	if err != nil {
		//return transaction, false, err
		transaction.TXMemo = "券不足"
		transaction.TXErrMsg = TXFrom + ":Payment > Balance"
		fmt.Printf("Payment: %s\n", Payment)
		fmt.Printf("balance: %s\n", balance)
		fmt.Printf("position: %s\n", position)
		return transaction, false, TXFrom + ":Payment > Balance"
	}
	transaction.TXFromBalance = balance
	transaction.TXFromPosition = position
	transaction.TXHcode = TXID
	transaction.TXStatus = "Pending"

	err2 := updateTransactionTXHcode(stub, TXID, TXHcode)
	if err2 != nil {
		//return transaction, false, err2
		transaction.TXMemo = "更正失敗"
		transaction.TXErrMsg = TXID + ":updateTransactionTXHcode execution failed."
		return transaction, false, TXID + ":updateTransactionTXHcode execution failed."
	}

	return transaction, isPutToQueue, ""

}

func updateEndDayTransactionStatus(stub shim.ChaincodeStubInterface, TXID string) (string, error) {
	var MatchedTXID string
	MatchedTXID = ""
	TimeNow := time.Now().Format(timelayout)
	transaction, err := getTransactionStructFromID(stub, TXID)
	if transaction.TXStatus != "Pending" && transaction.TXStatus != "Waiting4Payment" {
		return MatchedTXID, errors.New("Failed to find Transaction Pending OR Waiting4Payment TXStatus")
	}
	TXStatus := transaction.TXStatus
	TXMemo := ""
	if TXStatus == "Waiting4Payment" {
		TXMemo = "同資回報系統錯誤"
	}
	if TXStatus == "Pending" {
		TXMemo = "日終尚未比對取消"
	}

	transaction.TXStatus = "Cancelled"
	transaction.TXMemo = TXMemo
	transaction.UpdateTime = TimeNow
	transactionAsBytes, err := json.Marshal(transaction)
	if err != nil {
		return MatchedTXID, err
	}
	err = stub.PutState(TXID, transactionAsBytes)
	if err != nil {
		return MatchedTXID, err
	}
	if TXStatus == "Waiting4Payment" {
		MatchedTXID = transaction.MatchedTXID
		TXMemo = "同資回報系統錯誤"
		transaction2, _ := getTransactionStructFromID(stub, MatchedTXID)
		if transaction2 != nil {
			transaction2.TXStatus = "Cancelled"
			transaction2.TXMemo = TXMemo
			transaction2.UpdateTime = TimeNow
			transaction2AsBytes, err := json.Marshal(transaction2)
			if err != nil {
				return MatchedTXID, err
			}
			err = stub.PutState(MatchedTXID, transaction2AsBytes)
			if err != nil {
				return MatchedTXID, err
			}
		}

		TXType := transaction.TXType
		SecurityID := transaction.SecurityID
		SecurityAmount := transaction.SecurityAmount
		Payment := transaction.Payment
		TXFrom := transaction.TXFrom
		TXTo := transaction.TXTo
		//BankFrom := SubString(transaction.TXFrom, 0, 3)
		//BankTo := SubString(transaction.TXTo, 0, 3)

		if TXType == "S" {
			senderBalance, receiverBalance, err := resetAccountBalance(stub, SecurityID, SecurityAmount, Payment, TXFrom, TXTo)
			senderBalance, receiverBalance, err = resetSecurityAmount(stub, SecurityID, Payment, TXFrom, TXTo)
			err = updateBankTotals(stub, TXFrom, SecurityID, Payment, false)
			if err != nil {
				return MatchedTXID, err
			}
			err = updateBankTotals(stub, TXTo, SecurityID, Payment, true)
			if err != nil {
				return MatchedTXID, err
			}
			if (senderBalance < 0) || (receiverBalance < 0) {
				return MatchedTXID, errors.New("senderBalance,receiverBalance <0")
			}

		}
		if TXType == "B" {
			senderBalance, receiverBalance, err := resetAccountBalance(stub, SecurityID, SecurityAmount, Payment, TXTo, TXFrom)
			senderBalance, receiverBalance, err = resetSecurityAmount(stub, SecurityID, Payment, TXTo, TXFrom)
			err = updateBankTotals(stub, TXFrom, SecurityID, Payment, false)
			if err != nil {
				return MatchedTXID, err
			}
			err = updateBankTotals(stub, TXTo, SecurityID, Payment, true)
			if err != nil {
				return MatchedTXID, err
			}
			if (senderBalance < 0) || (receiverBalance < 0) {
				return MatchedTXID, errors.New("senderBalance,receiverBalance <0")
			}

		}
	}

	return MatchedTXID, nil
}

func updateEndDayQueuedTransactionStatus(stub shim.ChaincodeStubInterface, TXKEY string, TXID string, MatchedTXID string) error {

	TimeNow := time.Now().Format(timelayout)
	queuedTX, err := getQueueStructFromID(stub, TXKEY)
	if err != nil {
		return err
	}

	var doflg bool
	doflg = false

	for key, val := range queuedTX.TXIDs {
		fmt.Printf("qkey1: %d\n", key)
		fmt.Printf("qval1: %s\n", val)

		TXStatus := queuedTX.Transactions[key].TXStatus
		TXMemo := ""
		if TXStatus == "Waiting4Payment" {
			TXMemo = "同資回報系統錯誤"
		}
		if TXStatus == "Pending" {
			TXMemo = "日終尚未比對取消"
		}

		if val == TXID {
			fmt.Printf("qkey2: %d\n", key)
			fmt.Printf("qval2: %s\n", val)

			if (queuedTX.Transactions[key].TXStatus == "Pending") || (queuedTX.Transactions[key].TXStatus == "Waiting4Payment") {
				queuedTX.Transactions[key].TXStatus = "Cancelled"
				queuedTX.Transactions[key].TXMemo = TXMemo
				queuedTX.Transactions[key].UpdateTime = TimeNow
				doflg = true
			}
		}
		if val == MatchedTXID {
			fmt.Printf("qkey3: %d\n", key)
			fmt.Printf("qval3: %s\n", val)

			if (queuedTX.Transactions[key].TXStatus == "Pending") || (queuedTX.Transactions[key].TXStatus == "Waiting4Payment") {
				queuedTX.Transactions[key].TXStatus = "Cancelled"
				queuedTX.Transactions[key].TXMemo = TXMemo
				queuedTX.Transactions[key].UpdateTime = TimeNow
				doflg = true
			}
		}
	}
	if doflg != true {
		return errors.New("Failed to find Queued Pending OR Waiting4Payment TXStatus ")
	}

	queuedAsBytes, err := json.Marshal(queuedTX)
	if err != nil {
		return err
	}
	err = stub.PutState(TXKEY, queuedAsBytes)
	if err != nil {
		return err
	}
	return nil
}

func updateEndDayHistoryTransactionStatus(stub shim.ChaincodeStubInterface, HTXKEY string, TXID string, MatchedTXID string) error {

	TimeNow := time.Now().Format(timelayout)
	historyTX, err := getHistoryTransactionStructFromID(stub, HTXKEY)
	if err != nil {
		return err
	}

	var doflg bool
	doflg = false

	for key, val := range historyTX.TXIDs {
		fmt.Printf("hkey1: %d\n", key)
		fmt.Printf("hval1: %s\n", val)

		TXStatus := historyTX.Transactions[key].TXStatus
		TXMemo := ""
		if TXStatus == "Waiting4Payment" {
			TXMemo = "同資回報系統錯誤"
		}
		if TXStatus == "Pending" {
			TXMemo = "日終尚未比對取消"
		}

		if val == TXID {
			fmt.Printf("hkey2: %d\n", key)
			fmt.Printf("hval2: %s\n", val)
			if historyTX.Transactions[key].TXStatus == "Pending" || historyTX.Transactions[key].TXStatus == "Waiting4Payment" {
				historyTX.Transactions[key].TXStatus = "Cancelled"
				historyTX.Transactions[key].TXMemo = TXMemo
				historyTX.Transactions[key].UpdateTime = TimeNow
				historyTX.TXStatus[key] = "Cancelled"
				doflg = true
			}
		}
		if val == MatchedTXID {
			fmt.Printf("hkey3: %d\n", key)
			fmt.Printf("hval3: %s\n", val)
			if historyTX.Transactions[key].TXStatus == "Pending" || historyTX.Transactions[key].TXStatus == "Waiting4Payment" {
				historyTX.Transactions[key].TXStatus = "Cancelled"
				historyTX.Transactions[key].TXMemo = TXMemo
				historyTX.Transactions[key].UpdateTime = TimeNow
				historyTX.TXStatus[key] = "Cancelled"
				doflg = true
			}
		}
	}
	if doflg != true {
		return errors.New("Failed to find History Pending OR Waiting4Payment TXStatus ")
	}

	historyAsBytes, err := json.Marshal(historyTX)
	if err != nil {
		return err
	}
	err = stub.PutState(HTXKEY, historyAsBytes)
	if err != nil {
		return err
	}
	return nil
}

//peer chaincode query -n mycc -c '{"Args":["queryTXIDTransactions", "BANK002B00200000000120180408050918"]}' -C myc
//peer chaincode query -n mycc -c '{"Args":["queryTXIDTransactions", "BANK002B00200000000120180408051245"]}' -C myc

func (s *SmartContract) queryTXIDTransactions(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	NewTXAsBytes, _ := APIstub.GetState(args[0])
	NewTX := Transaction{}
	json.Unmarshal(NewTXAsBytes, &NewTX)

	NewTXAsBytes, err := json.Marshal(NewTX)
	if err != nil {
		return shim.Error("Failed to query NewTX state")
	}

	return shim.Success(NewTXAsBytes)
}

//peer chaincode query -n mycc -c '{"Args":["queryTXKEYTransactions", "20180408"]}' -C myc

func (s *SmartContract) queryTXKEYTransactions(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	QueuedTXAsBytes, _ := APIstub.GetState(args[0])
	QueuedTX := QueuedTransaction{}
	json.Unmarshal(QueuedTXAsBytes, &QueuedTX)

	QueuedTXAsBytes, err := json.Marshal(QueuedTX.Transactions)
	if err != nil {
		return shim.Error("Failed to query QueuedTX state")
	}

	return shim.Success(QueuedTXAsBytes)
}

//peer chaincode query -n mycc1 -c '{"Args":["queryHistoryTXKEYTransactions", "H20180408"]}' -C myc
func (s *SmartContract) queryHistoryTXKEYTransactions(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	HistoryNewTXAsBytes, _ := APIstub.GetState(args[0])
	HistoryNewTX := TransactionHistory{}
	json.Unmarshal(HistoryNewTXAsBytes, &HistoryNewTX)

	HistoryNewTXAsBytes, err := json.Marshal(HistoryNewTX.Transactions)
	if err != nil {
		return shim.Error("Failed to query HistoryNewTX state")
	}

	return shim.Success(HistoryNewTXAsBytes)
}

//peer chaincode query -n mycc -c '{"Args":["getHistoryForTransaction", "BANK002B00200000000120180408050918"]}' -C myc

func (s *SmartContract) getHistoryForTransaction(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	TXID := args[0]

	fmt.Printf("- start getHistoryForTransaction: %s\n", TXID)

	resultsIterator, err := APIstub.GetHistoryForKey(TXID)
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

	fmt.Printf("- getHistoryForTransaction returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc -c '{"Args":["getHistoryTXIDForTransaction","BK004S00400000000120180610041355","9476d9983cd1914d0c041b810d99dbbeee9f710bd03ee73ba71ff6770dc34b7a"]}' -C myc
func (s *SmartContract) getHistoryTXIDForTransaction(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	TransactionID := args[0]
	TXID := args[1]

	fmt.Printf("- start getHistoryTXIDForTransaction: %s\n", TransactionID)
	fmt.Printf("- start getHistoryTXIDForTransaction: %s\n", TXID)

	resultsIterator, err := APIstub.GetHistoryForKey(TransactionID)
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

	fmt.Printf("- getHistoryTXIDForTransaction returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc3 -c '{"Args":["getHistoryForQueuedTransaction", "H20180415"]}' -C myc

func (s *SmartContract) getHistoryForQueuedTransaction(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	TXKEY := args[0]

	fmt.Printf("- start getHistoryForQueuedTransaction: %s\n", TXKEY)

	resultsIterator, err := APIstub.GetHistoryForKey(TXKEY)
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

	fmt.Printf("- getHistoryForQueuedTransaction returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc -c '{"Args":["getHistoryTXIDForQueuedTransaction","20180610","a4723f60d5c85d29a2107382fb8e3c8c1624924b970efa04f313727a0dfaa0ff"]}' -C myc
func (s *SmartContract) getHistoryTXIDForQueuedTransaction(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	QueuedTransactionID := args[0]
	TXID := args[1]

	fmt.Printf("- start getHistoryTXIDForQueuedTransaction: %s\n", QueuedTransactionID)
	fmt.Printf("- start getHistoryTXIDForQueuedTransaction: %s\n", TXID)

	resultsIterator, err := APIstub.GetHistoryForKey(QueuedTransactionID)
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

	fmt.Printf("- getHistoryTXIDForQueuedTransaction returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc -c '{"Args":["queryAllTransactions", "BANK002B00200000000120180408050918","BANK002B00200000000120180408051246"]}' -C myc

func (s *SmartContract) queryAllTransactions(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}
	//TXID = BankFrom + TXType + TXFrom + TimeNow
	//BANK002B00200000000120180406143001
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

//peer chaincode query -n mycc -c '{"Args":["queryAllQueuedTransactions", "20180408","20180409"]}' -C myc

func (s *SmartContract) queryAllQueuedTransactions(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}
	//TXKEY = SubString(TimeNow,0,8)
	//20180406
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

//peer chaincode query -n mycc3 -c '{"Args":["queryAllHistoryTransactions", "20180415","20180416"]}' -C myc -v 1.0
func (s *SmartContract) queryAllHistoryTransactions(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}
	//TXKEY = SubString(TimeNow,0,8)
	//20180406
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

//peer chaincode query -n mycc -c '{"Args":["queryAllTransactionKeys", "BANK002" , "BANK009"]}' -C myc
func (s *SmartContract) queryAllTransactionKeys(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

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

//peer chaincode query -n mycc -c '{"Args":["queryQueuedTransactionStatus","1070609","Finished"]}' -C myc
func (s *SmartContract) queryQueuedTransactionStatus(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}
	TXKEY := args[0]
	TXStatus := args[1]

	QueuedAsBytes, _ := APIstub.GetState(TXKEY)
	QueuedTX := QueuedTransaction{}
	json.Unmarshal(QueuedAsBytes, &QueuedTX)

	var doflg bool
	doflg = false
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"TXKEY\":")
	buffer.WriteString("\"")
	buffer.WriteString(QueuedTX.TXKEY)
	buffer.WriteString("\"")

	for key, val := range QueuedTX.Transactions {
		if val.TXStatus == TXStatus {
			buffer.WriteString(", \"QueuedKey\":")
			buffer.WriteString(strconv.Itoa(key))
			buffer.WriteString(", \"TXID\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXID)
			buffer.WriteString(", \"TXType\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXType)
			buffer.WriteString(", \"TXFrom\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXFrom)
			buffer.WriteString(", \"TXTo\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXTo)
			buffer.WriteString(", \"BankFrom\":")
			buffer.WriteString(QueuedTX.Transactions[key].BankFrom)
			buffer.WriteString(", \"BankTo\":")
			buffer.WriteString(QueuedTX.Transactions[key].BankTo)
			buffer.WriteString(", \"SecurityID\":")
			buffer.WriteString(QueuedTX.Transactions[key].SecurityID)
			buffer.WriteString(", \"SecurityAmount\":")
			buffer.WriteString(strconv.FormatInt(QueuedTX.Transactions[key].SecurityAmount, 10))
			buffer.WriteString(", \"Payment\":")
			buffer.WriteString(strconv.FormatInt(QueuedTX.Transactions[key].Payment, 10))
			buffer.WriteString(", \"TXStatus\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXStatus)
			buffer.WriteString(", \"TXMemo\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXMemo)
			buffer.WriteString(", \"TXErrMsg\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXErrMsg)
			buffer.WriteString(", \"TXHcode\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXHcode)
			buffer.WriteString(", \"MatchedTXID\":")
			buffer.WriteString(QueuedTX.Transactions[key].MatchedTXID)
			buffer.WriteString(", \"CreateTime\":")
			buffer.WriteString(QueuedTX.Transactions[key].CreateTime)
			buffer.WriteString(", \"UpdateTime\":")
			buffer.WriteString(QueuedTX.Transactions[key].UpdateTime)
			buffer.WriteString(", \"TXIndex\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXIndex)
			buffer.WriteString(", \"TXSIndex\":")
			buffer.WriteString(QueuedTX.Transactions[key].TXSIndex)
			doflg = true
		}
	}
	if doflg != true {
		return shim.Error("Failed to find QueuedTransaction ")
	}
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())
}

//peer chaincode query -n mycc -c '{"Args":["queryHistoryTransactionStatus","H1070609","Finished"]}' -C myc
func (s *SmartContract) queryHistoryTransactionStatus(APIstub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}
	HTXKEY := args[0]
	TXStatus := args[1]

	HistoryAsBytes, _ := APIstub.GetState(HTXKEY)
	HistoryTX := TransactionHistory{}
	json.Unmarshal(HistoryAsBytes, &HistoryTX)

	var doflg bool
	doflg = false
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString("{\"HTXKEY\":")
	buffer.WriteString("\"")
	buffer.WriteString(HistoryTX.TXKEY)
	buffer.WriteString("\"")

	for key, val := range HistoryTX.TXStatus {
		if val == TXStatus {
			buffer.WriteString(", \"HistoryKey\":")
			buffer.WriteString(strconv.Itoa(key))
			buffer.WriteString(", \"TXID\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXID)
			buffer.WriteString(", \"TXType\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXType)
			buffer.WriteString(", \"TXKinds\":")
			buffer.WriteString(HistoryTX.TXKinds[key])
			buffer.WriteString(", \"TXFrom\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXFrom)
			buffer.WriteString(", \"TXTo\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXTo)
			buffer.WriteString(", \"BankFrom\":")
			buffer.WriteString(HistoryTX.Transactions[key].BankFrom)
			buffer.WriteString(", \"BankTo\":")
			buffer.WriteString(HistoryTX.Transactions[key].BankTo)
			buffer.WriteString(", \"SecurityID\":")
			buffer.WriteString(HistoryTX.Transactions[key].SecurityID)
			buffer.WriteString(", \"SecurityAmount\":")
			buffer.WriteString(strconv.FormatInt(HistoryTX.Transactions[key].SecurityAmount, 10))
			buffer.WriteString(", \"Payment\":")
			buffer.WriteString(strconv.FormatInt(HistoryTX.Transactions[key].Payment, 10))
			buffer.WriteString(", \"TXStatus\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXStatus)
			buffer.WriteString(", \"TXMemo\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXMemo)
			buffer.WriteString(", \"TXErrMsg\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXErrMsg)
			buffer.WriteString(", \"TXHcode\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXHcode)
			buffer.WriteString(", \"MatchedTXID\":")
			buffer.WriteString(HistoryTX.Transactions[key].MatchedTXID)
			buffer.WriteString(", \"CreateTime\":")
			buffer.WriteString(HistoryTX.Transactions[key].CreateTime)
			buffer.WriteString(", \"UpdateTime\":")
			buffer.WriteString(HistoryTX.Transactions[key].UpdateTime)
			buffer.WriteString(", \"TXIndex\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXIndex)
			buffer.WriteString(", \"TXSIndex\":")
			buffer.WriteString(HistoryTX.Transactions[key].TXSIndex)
			doflg = true
		}
	}
	if doflg != true {
		return shim.Error("Failed to find TransactionHistory ")
	}
	buffer.WriteString("}")
	buffer.WriteString("]")
	fmt.Printf("%s", buffer.String())

	return shim.Success(buffer.Bytes())
}

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	eos "github.com/eoscanada/eos-go"
)

const stdTime string = "2006-01-02 15:04:05"

//用来接json
type GetTableRspItem struct {
	Hash       string `json:"hash"`
	Questioner string `json:"questioner"`
	CreateTime string `json:"create_time"`
}

//问题结构体
type Question struct {
	Hash       uint64
	Questioner string
	CreateTime time.Time
}

//提问参数
type QuestParam struct {
	Questioner eos.AccountName `json:"questioner"`
	Question   string          `json:"question"`
}

func rspItem2Question(item *GetTableRspItem) (*Question, error) {
	hash, err := strconv.ParseUint(item.Hash, 10, 64)
	if err != nil {
		fmt.Println("rspItem2Question err:", err)
		return nil, err
	}
	t, err := time.Parse(stdTime, strings.ReplaceAll(item.CreateTime, "T", " ")) //go不支持eos的时间格式
	if err != nil {
		fmt.Println("rspItem2Question err:", err)
		return nil, err
	}
	return &Question{
		Hash:       uint64(hash),
		Questioner: item.Questioner,
		CreateTime: t,
	}, nil
}

func main() {
	ctx, _ := context.WithCancel(context.Background())
	api := eos.New("http://127.0.0.1:8888/")

	acc := eos.AccountName("t1") //账号名

	infoRsp, err := api.GetInfo(ctx)
	if err != nil {
		fmt.Println("GetInfo err:", err)
		return
	}
	fmt.Println("infoRsp:", infoRsp)

	accRsp, err := api.GetAccount(ctx, acc)
	if err != nil {
		fmt.Println("GetAccount err1:", err)
		return
	}
	fmt.Println("accRsp:", accRsp)

	pubkeys := accRsp.Permissions[0].RequiredAuth.Keys
	fmt.Println("pubkeys:", pubkeys, "\n")

	api.SetSigner(eos.NewKeyBag())

	//看表
	fmt.Println("提问前>>")
	watchTable(ctx, api)

	//提问
	privKey := "5JCPMjRHmgtbk1v9Z8zHFWDoA5sML7yWFXBQXYvkAecgrLARHoF"
	ask(ctx, api, privKey)

	//再看表
	fmt.Println("提问后>>")
	watchTable(ctx, api)
}

//看表
func watchTable(ctx context.Context, api *eos.API) {
	//先看表
	getTableReq := eos.GetTableRowsRequest{
		JSON:  true,        //是否启用json
		Code:  "t1",        //合约名字
		Scope: "t1",        //scope（填合约部署者名字）
		Table: "questions", //表名
		Index: "1",         //默认1
		Limit: 10,          //限制条数
	}
	data, err := api.GetTableRows(ctx, getTableReq)
	if err != nil {
		fmt.Println("GetTableRows err:", err)
		return
	}
	var tmp = []GetTableRspItem{}
	if err := data.JSONToStructs(&tmp); err != nil {
		fmt.Println("unmarshal getTableRsp err:", err)
		return
	}
	questions := []*Question{}
	for _, item := range tmp {
		if q, err := rspItem2Question(&item); err == nil {
			questions = append(questions, q)
			fmt.Println("问题hash:", q.Hash)
			fmt.Println("提问者:", q.Questioner)
			fmt.Println("时间:", q.CreateTime)
		}
	}
}

//提问
func ask(ctx context.Context, api *eos.API, privKey string) {
	act := &eos.Action{
		Account: eos.AccountName("t1"),
		Name:    eos.ActionName("question"),
		Authorization: []eos.PermissionLevel{
			{
				Actor:      eos.AccountName("t1"),
				Permission: eos.PermissionName("active"),
			},
		},
		ActionData: eos.NewActionData(QuestParam{
			Questioner: eos.AccountName("t1"),
			Question:   "QmaYQaCwxg8pK9Z5QQC8vrb1hdBC5nV8YFGwGNVNQ3fWka",
		}),
	}
	api.Signer.ImportPrivateKey(ctx, privKey)
	_, err := api.SignPushActions(ctx, act)
	if err != nil {
		fmt.Println("ask err:", err)
	}
}

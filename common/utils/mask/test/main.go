package main

import (
	"fmt"
	"strings"

	"github.com/wfusion/gofusion/common/utils/mask"
)

func main() {
	caller := "replace.your.caller"
	// remove NewEngine() outside for loop, and one Engine Object one thread/goroutin
	if eng, err := mask.NewEngine(caller); err == nil {
		eng.ApplyConfig(defaultCfg)
		fmt.Printf("DLP %s Demo:\n\n", eng.GetVersion())
		inStr := `我的邮件是abcd@abcd.com,
18612341234是我的电话
你家住在哪里啊? 我家住在北京市海淀区北三环西路43号,
mac地址 06-06-06-aa-bb-cc
收件人：张真人  手机号码：18612341234`
		if results, err := eng.Detect(inStr); err == nil {
			fmt.Printf("\t1. Detect( inStr: %s )\n", inStr)
			eng.ShowResults(results)
		}
		if outStr, _, err := eng.Deidentify(inStr); err == nil {
			fmt.Printf("\t2. Deidentify( inStr: %s )\n", inStr)
			fmt.Printf("\toutStr: %s\n", outStr)
			//eng.ShowResults(results)
			fmt.Println()
		}
		inStr = `18612341234`
		if outStr, err := eng.Mask(inStr, mask.CHINAPHONE); err == nil {
			fmt.Printf("\t3. Mask( inStr: %s )\n", inStr)
			fmt.Printf("\toutStr: %s\n", outStr)
			fmt.Println()
		}

		inMap := map[string]string{"nothing": "nothing", "uid": "10086", "k1": "my phone is 18612341234 and 18612341234"} // extract KV?

		if results, err := eng.DetectMap(inMap); err == nil {
			fmt.Printf("\t4. DetectMap( inMap: %+v )\n", inMap)
			eng.ShowResults(results)
		}

		fmt.Printf("\t5. DeidentifyMap( inMap: %+v )\n", inMap)
		if outMap, results, err := eng.DeidentifyMap(inMap); err == nil {
			fmt.Printf("\toutMap: %+v\n", outMap)
			eng.ShowResults(results)
			fmt.Println()
		}

		inJSON := `{"objList":[{"uid":"10086"},{"uid":"[\"aaaa\",\"bbbb\"]"}]}`

		if results, err := eng.DetectJSON(inJSON); err == nil {
			fmt.Printf("\t6. DetectJSON( inJSON: %s )\n", inJSON)
			eng.ShowResults(results)

			if outJSON, err := eng.DeidentifyJSONByResult(inJSON, results); err == nil {
				fmt.Printf("\t7. DeidentifyJSONByResult( inJSON: %s , results: %v )\n", inJSON, results)
				fmt.Printf("\toutJSON: %s\n", outJSON)
				eng.ShowResults(results)
				fmt.Println()
			}
		}

		if outJSON, results, err := eng.DeidentifyJSON(inJSON); err == nil {
			fmt.Printf("\t7. DeidentifyJSON( inJSON: %s )\n", inJSON)
			fmt.Printf("\toutJSON: %s\n", outJSON)
			eng.ShowResults(results)
			fmt.Println()
		}
		inStr = "abcd@abcd.com"
		maskRule := "EmailMaskRule01"
		if outStr, err := eng.Mask(inStr, maskRule); err == nil {
			fmt.Printf("\t8. Mask( inStr: %s , %s)\n", inStr, maskRule)
			fmt.Printf("\toutStr: %s\n", outStr)
			fmt.Println()
		}
		// Custom desensitization, Mailbox user name to retain the first and
		// last one character, retain all domain names
		eng.RegisterMasker("EmailMaskRule02", func(in string) (string, error) {
			list := strings.Split(in, "@")
			if len(list) >= 2 {
				prefix := list[0]
				domain := list[1]
				if len(prefix) > 2 {
					prefix = "*" + prefix[1:len(prefix)-1] + "*"
				} else if len(prefix) > 0 {
					prefix = "*" + prefix[1:]
				} else {
					return in, fmt.Errorf("%s is not Email", in)
				}
				ret := prefix + "@" + domain
				return ret, nil
			} else {
				return in, fmt.Errorf("%s is not Email", in)
			}
		})
		inStr = "abcd@abcd.com"
		maskRule = "EmailMaskRule02"
		if outStr, err := eng.Mask(inStr, maskRule); err == nil {
			fmt.Printf("\t9. Mask( inStr: %s , %s)\n", inStr, maskRule)
			fmt.Printf("\toutStr: %s\n", outStr)
			fmt.Println()
		}

		inStr = "loginfo:[ uid:10086, phone:18612341234]"
		if outStr, results, err := eng.Deidentify(inStr); err == nil {
			fmt.Printf("\t10. Detect( inStr: %s )\n", inStr)
			eng.ShowResults(results)
			fmt.Printf("\toutStr: %s\n", outStr)
			fmt.Println()
		}
		type EmailType string
		// For structures that require recursion,
		// you need to fill in 'mask: “Deep”' to recursively desensitize
		type Foo struct {
			Email         EmailType `mask:"EMAIL"`
			PhoneNumber   string    `mask:"CHINAPHONE"`
			Idcard        string    `mask:"CHINAID"`
			Buffer        string    `mask:"DEIDENTIFY"`
			EmailPtrSlice []*struct {
				Val string `mask:"EMAIL"`
			} `mask:"DEEP"`
			PhoneSlice []string `mask:"CHINAPHONE"`
			Extinfo    *struct {
				Addr string `mask:"ADDRESS"`
			} `mask:"DEEP"`
			EmailArray [2]string   `mask:"EMAIL"`
			NULLPtr    *Foo        `mask:"DEEP"`
			IFace      interface{} `mask:"ExampleTAG"`
		}
		var inObj = Foo{
			"abcd@abcd.com",
			"18612341234",
			"110225196403026127",
			"我的邮件是abcd@abcd.com",
			[]*struct {
				Val string `mask:"EMAIL"`
			}{{"3333@4444.com"}, {"5555@6666.com"}},
			[]string{"18612341234", ""},
			&struct {
				Addr string "mask:\"ADDRESS\""
			}{"北京市海淀区北三环西路43号"},
			[2]string{"abcd@abcd.com", "3333@4444.com"},
			nil,
			"abcd@abcd.com",
		}
		inPtr := &inObj
		inObj.NULLPtr = inPtr
		fmt.Printf("\t11. MaskStruct( inPtr: %+v, Extinfo: %+v)\n", inPtr, *(inPtr.Extinfo))
		// The MaskStruct parameter must be a pointer to modify the internal elements of the struct
		if outPtr, err := eng.MaskStruct(inPtr); err == nil {
			fmt.Printf("\toutObj: %+v, Extinfo:%+v\n", outPtr, inObj.Extinfo)
			fmt.Printf("\t\t EmailPtrSlice:\n")
			for i, ePtr := range inObj.EmailPtrSlice {
				fmt.Printf("\t\t\t[%d] = %s\n", i, ePtr.Val)
			}
			fmt.Println()
		} else {
			fmt.Println(err.Error())
		}
		eng.Close()
	} else {
		fmt.Println("[dlp] NewEngine error: ", err.Error())
	}
}

const (
	defaultCfg = `
# GODLP config file
# keys are UpperCamelCase, and are same as conf struct in conf/conf.go
Global:
  Date: 2021-10-27
  ApiVersion: v2
  Mode: release # debug|release
  AllowRPC:  false # true for remote service with rpc, false for pure client SDK, default is false
  # if EnableRules is empty, it means all rules are enabled, but if EnableRules contains some ruleIDs, only these rules are enabled.
  # Then DLP will remove some ruleIDs if they are in DisableRules.
  EnableRules: []
  # disable a certain rule by push ruleID in disableRules
  DisableRules: []
  MaxLogInput: 4096
  MaxRegexRuleID: 0
MaskRules:
  # Example MaskRule start
  - RuleName: ExampleCHAR # Name of MaskRule
    MaskType: CHAR # one of [CHAR, TAG, REPLACE, ALGO ]
    Value: "*"
    Offset: 1
    Padding: 0 # offset from the tail
    Length: 5
    Reverse: true
    IgnoreCharSet: "@"
    IgnoreKind: [ NUMERIC ] # one of [NUMERIC, ALPHA_UPPER_CASE, ALPHA_LOWER_CASE, WHITESPACE, PUNCTUATION]
  - RuleName: ALL
    MaskType: CHAR
    Value: "*"
  - RuleName: ExampleTAG
    MaskType: TAG
  - RuleName: ExampleREPLACE
    MaskType: REPLACE
    Value: "<REPLACED>"
  - RuleName: ExampleEMPTY
    MaskType: REPLACE
    Value: ""
  - RuleName: ExampleBASE64
    MaskType: ALGO
    Value: "BASE64" # one of [BASE64, MD5, CRC32, ADDRESS, NUMBER, DEIDENTIFY]
  - RuleName: ExampleMD5
    MaskType: ALGO
    Value: "MD5" # one of [BASE64, MD5, CRC32, ADDRESS, NUMBER, DEIDENTIFY]
  - RuleName: DEIDENTIFY
    MaskType: ALGO
    Value: "DEIDENTIFY"
  # Example MaskRule end
  - RuleName: "NULL"
    MaskType: REPLACE
    Value: "NULL"
  - RuleName: CHINAPHONE
    MaskType: CHAR
    Value: "*"
    Offset: 3
    Length: 6
    IgnoreCharSet: "-"
  - RuleName: PHONE
    MaskType: CHAR
    Value: "*"
    Offset: 2
    Padding: 2
    IgnoreCharSet: "-"
  - RuleName: CHINAID
    MaskType: CHAR
    Value: "*"
    Offset: 1
    Padding: 1
  - RuleName: IDCARD
    MaskType: CHAR
    Value: "*"
    Offset: 1
    Padding: 1
  - RuleName: EMAIL
    MaskType: CHAR
    Value: "*"
    Offset: 1
    IgnoreCharSet: "@"
  - RuleName: UID
    MaskType: CHAR
    Value: "*"
    Offset: 1
  - RuleName: BANK
    MaskType: CHAR
    Value: "*"
    Offset: 4
    Reverse: true
  - RuleName: PASSPORT
    MaskType: CHAR
    Value: "*"
    Offset: 2
    Padding: 2
  - RuleName: ADDRESS
    MaskType: ALGO
    Value: ADDRESS
  - RuleName: NAME
    MaskType: CHAR
    Value: "*"
    Offset: 3
  - RuleName: NUMBER
    MaskType: ALGO
    Value: NUMBER
  - RuleName: MACADDR
    MaskType: CHAR
    Value: "*"
    Reverse: true
    Length: 8
    IgnoreCharSet: ":-"
  - RuleName: ABA
    MaskType: CHAR
    Value: "*"
    Reverse: true
    Length: 6
    IgnoreCharSet: "-"
  - RuleName: BITCOIN
    MaskType: CHAR
    Value: "*"
    Length: 24
    Offset: 5
  - RuleName: CAR
    MaskType: CHAR
    Value: "*"
    Offset: 4
    Padding: 2
  - RuleName: DID
    MaskType: CHAR
    Value: "*"
    Offset: 4
    Padding: 4
    IgnoreCharSet: "-"
  - RuleName: BIRTH
    MaskType: CHAR
    Value: "*"
    Reverse: true
    Length: 2
    IgnoreCharSet: "-"
  - RuleName: AGE
    MaskType: CHAR
    Value: "*"
    Reverse: true
    Length: 1
  - RuleName: EDU
    MaskType: CHAR
    Value: "*"
    Padding: 6
  - RuleName: GODLP
    MaskType: REPLACE
    Value: "<GODLP Copyright 2021>"
# Rules are combined with defaultRules and privateRules. The default rules are managed by DLP team, do not modify defualtRules directly, disable a certain rule then copy and modify it in privateRules
# 0< defaultRules ruleID<10000 ,and 10000<= private ruleID
# greater RuleID will overwrite smaller RuleID result if detect position is the same
Rules:
  # defaultRule start
  - RuleID: 1
    InfoType: PHONE
    Description: 手机号
    EnName: telephone_number
    CnName: 电话号码
    Level: L4
    # Detect feild is an array for detect methods, the relation of each item in detect is OR relation.
    # Regex: regex expression, no need to escape
    # KReg,VReg,KDict,VDict
    # Dict: [ word1, word2, ...],
    # (KReg || KDict) && (VReg || VDict)
    Detect:
      KReg: []
      KDict: []
      VReg:
        - 1(?:(((3[0-9])|(4[5-9])|(5[0-35-9])|(6[2,5-7])|(7[0135-8])|(8[0-9])|(9[0-35-9]))[ -]?\d{4}[ -]?\d{4})|((74)[ -]?[0-5]\d{3}[ -]?\d{4}))\b
      VDict: []
    # Filter contains blacklist.
    Filter:
      BAlgo: [MASKED] # supports MASKED, if detected value contains *, the result will not be returned
      BDict: [] # if one of results in blacklist dict, the result will not be returned.
      BReg: [] # blacklist regex list
    # Context: [word1, word2] one of context words has to be arround the result within ContextVerifyRange
    Verify:
      CDict: ["contact_phone", "remark_mobiles","ContactPhone", "phone","phones","number","telephone","telephones","cell","mobile","office","call","cellphone","cellphones","smartphone","smartphones","num","no","tel","linktel","contact","contactinfo","phoneno","phonenum","phonenumber","telephone_no","telephoneno","telephonenum","telephonenumber","mobilephoneno","mobliephonenum","mobilephonenumber","mobileno","moblieenum","mobilenumber","mobilecode","手机号","传真","手机","号码","联系","电话" ]
      CReg: []  # Regex list for context
      VAlgo: [] # value will be verified by verify function, such as IDCARD, 身份证校验函数
    Mask: CHINAPHONE # MaskRules.RuleName
    ExtInfo: # extra information, kv formate
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 2
    InfoType: EMAIL
    Description: 电子邮件地址
    EnName: EMAIL_address
    CnName: 电子邮箱
    GroupName: user_data
    Level: L4
    Detect:
      VReg:
        - \b(((([*+\-=?^_{|}~\w])|([*+\-=?^_{|}~\w][*+\-=?^_{|}~\.\w]{0,}[*+\-=?^_{|}~\w]))[@]\w+([-.]\w+)*\.[A-Za-z]{2,8}))\b
    Filter:
      BAlgo: [MASKED]
    Verify:
      VAlgo: [DOMAIN]
    Mask: EMAIL
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 4
    InfoType: CHINA_IDCARD
    Description: 中国身份证，只支持18位，需要通过校验算法
    EnName: china_id_card
    CnName: 中国身份证
    Level: L4
    Detect:
      VReg:
        - \b((1[1-5]|2[1-3]|3[1-7]|4[1-6]|5[0-4]|6[1-5]|[7-9]1)\d{4}(18|19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx])\b
    Verify:
      VAlgo: [IDCARD]
    Mask: CHINAID
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 5
    InfoType: DEBIT_CARD
    Description: 借记卡号,银联
    EnName: debit_card_account_number
    CnName: 借记卡号
    GroupName: user_data
    Level: L4
    Detect:
      VReg:
        - \b62\d{11,17}\b
    Verify:
      CDict: ["debit","card","visa debit","unionpay","借记卡"]
    Mask: BANK
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 6
    InfoType: CREDIT_CARD
    Description: 信用卡号
    EnName: credit_card_account_number
    CnName: 信用卡号
    Level: L4
    Detect:
      VReg:
        - \b((([1-9]\d{3})[\s-](\d{4})[\s-](\d{4})[\s-](\d{4})))\b
        - \b62\d{11, 14}\b
        - \b[1-9]\d{12,18}\b
    Verify:
      VAlgo: [CREDITCARD]
      CDict: [ "credit",
               "card",
               "visa",
               "unionpay",
               "mastercard",
               "amex",
               "discover",
               "jcb",
               "diners",
               "maestro",
               "instapayment",
               "信用卡"
      ]
    Mask: BANK
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 7
    InfoType: CHINA_PASSPORT
    Description: 护照,中国护照
    EnName: passport
    CnName: 护照
    Level: L4
    Detect:
      VReg:
        - \b(((1[45]\d{7})|([P|p|S|s]\d{7})|([S|s|G|g|E|e]\d{8})|([Gg|Tt|Ss|Ll|Qq|Dd|Aa|Ff]\d{8})|([H|h|M|m]\d{8,10})))\b
    Verify:
      CDict: ["passport","passport#","travel","document","book","bookid","catalog","citizenship","护照","证件","签证"]
    Mask: PASSPORT
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 8
    InfoType: ADDRESS
    Description: 中文地址
    EnName: address_cn
    CnName: 中文地址
    GroupName: user_data
    Level: L1
    Detect:
      VReg:
        - ((.{1,6}(区|镇)?.{1,6}(路|街).{1,6}号.{1,6}号楼.{1,6}单元.{1,6}(层|室|户)?)|(.{1,6}县.{1,6}(镇|乡)?.{1,6}(路|街).{1,6}号.{1,6}号楼.{1,6}单元.{1,6}(层|室|户)?)|(.{1,6}(区|镇)?.{1,6}小区.{1,6}号楼.{1,6}单元.{1,6}(层|室|户)?)|(.{1,6}县.{1,6}(镇|乡)?.{1,6}小区.{1,6}号楼.{1,6}单元.{1,6}(层|室|户)?)|(.{1,6}(路|街|里).{1,6}号.{1,6}(层|室|户)?)|(.{1,6}(镇|乡).{1,6}村.{1,6}(组|屯).{1,6}号?)|(.{1,6}(镇|乡|街).{1,6}(村|屯).{1,6}(组|号)?)|((.{1,6}省)?.{1,6}市.{1,6}(区|街|路).{1,6}(家园|里).{1,6}号))
        - ((.{2,6}?(省|自治区))|(.{1,6}?(市|自治区|自治州))|(.{1,6}?(县|区|镇|乡))){1,3}((.{1,6}(路|街|里|街道|村|屯|组))|(.{1,6}(小区|大厦|号|广场))){1,3}((.{1,6}(号楼))|(.{1,6}(单元))|(.{1,6}(层|室|户|号|房))|(\d+-\d+-\d+)){0,3}
    Verify:
      VAlgo: []
    Mask: ADDRESS
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 9
    InfoType: NAME
    Description: 人名
    EnName: name
    CnName: 人名
    GroupName: user_data
    Level: L4
    Detect:
      KDict: ["收件人"]
    Verify:
      VAlgo: []
    Mask: NAME
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 10
    InfoType: MACADDR
    Description: MAC地址
    EnName: MAC_address
    CnName: MAC地址
    GroupName: user_data
    Level: L3
    Detect:
      VReg:
        - \b[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}\b
        - \b([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})\b
    #Verify:  # MacAddr do not use Context verify yet
    #  CDict: ["ether","mac", "address", "地址", "macaddr", "macaddress", "addr", "mc"]
    Mask: MACADDR
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 11
    InfoType: ADDRESS
    Description: 中文地址,根据key来识别
    EnName: address_cn
    CnName: 中文地址
    Level: L4
    Detect:
      KDict:
        - 联系地址
    Mask: ADDRESS
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 12
    InfoType: ABA_ROUTING
    Description: ABA ROUTING 号码，支票底部的编码
    EnName: bank_branch_code
    CnName: 银行分支机构号
    Level: L4
    Detect:
      VReg:
        - \b([0123678]\d{8})\b
        - \b([0123678]\d{3}-\d{4}-\d)\b
    Verify:
      VAlgo: ["ABAROUTING"]
      CDict: [ "bank_branch_code","支行代码","aba","routing"]
    Mask: ABA
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 13
    InfoType: CHINA_DRIVER_LICENSE
    Description: 中国驾照，规则按身份证验证
    EnName: driving_license
    CnName: 中国驾照
    Level: L4
    Detect:
      VReg:
        - \b((1[1-5]|2[1-3]|3[1-7]|4[1-6]|5[0-4]|6[1-5]|[7-9]1)\d{4}(18|19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx])\b
    Verify:
      CDict: ["driver", "license", "driving", "驾驶证", "驾照"]
      VAlgo: [IDCARD]
    Mask: CHINAID
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 14
    InfoType: BITCOIN
    Description: 比特币钱包地址
    EnName: bitcoin
    CnName: 比特币钱包地址
    Level: L4
    Detect:
      VReg:
        - \b[13][a-km-zA-HJ-NP-Z1-9]{26,33}\b
    Verify:
      VAlgo: [BITCOIN]
    Mask: ExampleMD5
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 15
    InfoType: DOMAIN
    Description: 域名
    EnName: domain_name
    CnName: 域名
    Level: L1
    Detect:
      VReg:
        - \b((((([a-zA-Z0-9])|([a-zA-Z0-9][a-zA-Z0-9\-]{0,86}[a-zA-Z0-9]))\.(([a-zA-Z0-9])|([a-zA-Z0-9][a-zA-Z0-9\-]{0,73}[a-zA-Z0-9]))\.(([a-zA-Z0-9]{2,12}\.[a-zA-Z]{2,12})|([a-zA-Z]{2,25})))|((([a-zA-Z0-9])|([a-zA-Z0-9][a-zA-Z0-9\-]{0,162}[a-zA-Z0-9]))\.(([a-zA-Z0-9]{2,12}\.[a-zA-Z]{2,12})|([a-zA-Z]{2,25})))))\b
    Verify:
      VAlgo: [DOMAIN]
    #Mask:  # domain will not be masked,but as a result item
    ExtInfo:
      EnGroup: group_data
      CnGroup: 集团数据
  - RuleID: 16
    InfoType: IP
    Description: IP地址，包含v4和v6
    EnName: IP_address
    CnName: IP地址
    Level: L3
    Detect:
      VReg:
        - \b((?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?))\b
        - \b(?:(\s|\A))((([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|fe80:(:|(:[0-9a-fA-F]{1,4}){0,4})%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:)|:((:[0-9a-fA-F]{1,4}){1,7}|:))(?:(\s|\z))\b
    #Mask:  # IP will not be masked,but as a result item
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 17
    InfoType: US_PASSPORT
    Description: 美国护照
    EnName: us_passport
    CnName: 美国护照
    Level: L4
    Detect:
      VReg:
        - \b[0-9]{9}\b
    Verify:
      CDict: ["USA","passport","passport#","travel","document","book","bookid","catalog","citizenship","护照","证件","签证"]
    Mask: PASSPORT
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 18
    InfoType: US_BANK_NUMBER
    Description: 美国银行账号
    EnName: us_bank_account_number
    CnName: 美国银行账号
    Level: L4
    Detect:
      VReg:
        - \b[0-9]{8,17}\b
    Verify:
      CDict: ["usa","bank","check","account","account#","acct","save","debit"]
    Mask: BANK
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 19
    InfoType: US_ITIN
    Description: 美国纳税人识别号
    EnName: taxpayer_identification_number
    CnName: 美国纳税人识别号
    Level: L4
    Detect:
      VReg:
        - \b((9\d{2})((7[0-9]{1}|8[0-8]{1})|(9[0-2]{1})|(9[4-9]{1}))(\d{4}))\b
        - \b((9\d{2})[- ]{1}((7[0-9]{1}|8[0-8]{1})|(9[0-2]{1})|(9[4-9]{1}))[- ]{1}(\d{4}))\b
    Verify:
      CDict: ["individual","taxpayer","itin","tax","payer","taxid","tin"]
    Mask: BANK
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 20
    InfoType: PHONE
    Description: 美国电话号码
    EnName: us_telephone_number
    CnName: 美国电话号码
    Level: L4
    Detect:
      VReg:
        - \b((\d{3})\s*\d{3}[-\.\s]??\d{4}|\d{3}[-\.\s]\d{3}[-\.\s]\d{4})\b
        - \b(\d{3}[-\.\s]\d{3}[-\.\s]??\d{4})\b
    Verify:
      CDict: ["phone","phones","number","telephone","telephones","cell","mobile","office","call","cellphone","cellphones","smartphone","smartphones","num","no","tel","linktel","contact","contactinfo","phoneno","phonenum","phonenumber","telephone_no","telephoneno","telephonenum","telephonenumber","mobilephoneno","mobliephonenum","mobilephonenumber","mobileno","moblieenum","mobilenumber","mobilecode","手机号","传真","手机","号码","联系","电话"]
    Mask: PHONE
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 21
    InfoType: SG_NRIC_FIN
    Description: 新加坡身份证
    EnName: sg_id_card
    CnName: 新加坡身份证
    Level: L4
    Detect:
      VReg:
        - \b((?i)([STFG][0-9]{7}[A-Z]))\b
    Verify:
      CDict: ["fin","fin#","nric","nric#"]
    Mask: IDCARD
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 23
    InfoType: BANK_NUMBER
    Description: 银行账号,kv类型
    EnName: bank_account_number
    CnName: 银行账号
    Level: L4
    Detect:
      KDict: ["bankcard","bank_card","bank_account_number","银行卡","银行账号","信用卡","debit card","bank_account_no"]
    Mask: BANK
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 24
    InfoType: CAR_NUMBER
    Description: 车牌号,kv类型
    EnName: license_plate_number
    CnName: 车牌号
    Level: L4
    Detect:
      KDict: ["license_plate","car_number","车牌","licenseplate"]
    Mask: CAR
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 25
    InfoType: PASSPORT
    Description: 护照号,kv类型
    EnName: passport
    CnName: 护照号
    Level: L4
    Detect:
      KDict: ["passport","护照","港澳通行证","台湾通行证"]
      VReg:
        - \b(((1[45]\d{7})|([P|p|S|s]\d{7})|([S|s|G|g|E|e]\d{8})|([Gg|Tt|Ss|Ll|Qq|Dd|Aa|Ff]\d{8})|([H|h|M|m]\d{8,10})))\b
    Mask: PASSPORT
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 26
    InfoType: DID
    Description: 设备ID,kv类型
    EnName: did
    CnName: 设备ID
    Level: L3
    Detect:
      KDict: ["did","deviceid","device_id","did","deviceid","device_id"]
    Mask: DID
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 27
    InfoType: NAME
    Description: 姓名,kv类型
    EnName: name
    CnName: 姓名
    Level: L1
    Detect:
      KDict:  ["name","姓名","名字","sale_name","用户名"]
    Mask: NAME
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 28
    InfoType: BIRTHDAY
    Description: 生日,kv类型
    EnName: birthday
    CnName: 生日
    Level: L1
    Detect:
      KDict:  ["birthday","生日","birth","星座"]
    Mask: BIRTH
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 29
    InfoType: AGE
    Description: 年龄,kv类型
    EnName: age
    CnName: 年龄
    Level: L1
    Detect:
      KDict: ["age","年龄","岁数"]
      VReg:
        - \b(((1[0-5])|[1-9])?\d)\b
    Mask: AGE
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 30
    InfoType: EDUCATION
    Description: 学历,kv类型
    EnName: education_experience
    CnName: 学历
    Level: L3
    Detect:
      KDict: ["education","学历","educational background","学院","专业"]
    Mask: EDU
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 31
    InfoType: NATIONALITY
    Description: 国籍,kv类型
    EnName: nationality
    CnName: 国籍
    Level: L1
    Detect:
      KDict: ["nationality","国籍"]
    Mask: ExampleTAG
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 32
    InfoType: SSN
    Description: 社会保险卡,kv类型
    EnName: social_insurance_card
    CnName: 社会保险卡
    Level: L4
    Detect:
      KDict: ["社保号","SSN","soucial security number","社会保险号"]
    Mask: IDCARD
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 33
    InfoType: LOCATION
    Description: 经纬度信息,kv类型
    EnName: latitude_and_longitude_information
    CnName: 经纬度信息
    Level: L1
    Detect:
      KDict: ["latitude","longitude","lat","lng","经度","东经","西经","纬度","南纬","北纬"]
    Mask: ExampleTAG
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 34
    InfoType: GODLP
    Description: GODLP
    EnName: GODLP
    CnName: GODLP
    Level: L4
    Detect:
      VDict: ["4347cd408c1bd336a801867d30aace60","4f7738df4e3519d860e4554a4ca26d50"] # md5("GODLP")
    Mask: GODLP
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 35
    InfoType: PHONE
    Description: 电话，kv类型
    EnName: telephone_number
    CnName: 电话号码
    Level: L4
    Detect:
      KDict: ["电话","投诉电话","mobile","phone"]
    Filter:
      BAlgo: [MASKED]
    Mask: PHONE
    ExtInfo: # extra information, kv formate
      EnGroup: user_data
      CnGroup: 用户数据
  - RuleID: 36
    InfoType: UID
    Description: 用户user id
    EnName: userid
    CnName: 用户user_id
    GroupName: user_data
    Level: L3
    Detect:
      KDict: [ uid,user_id]
    Mask: UID
    ExtInfo:
      EnGroup: user_data
      CnGroup: 用户数据
# defaultRule end
`
)

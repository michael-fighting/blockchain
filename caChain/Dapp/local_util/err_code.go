package local_util

type ErrCode uint32

// 通用错误
const (
	ErrorParamWrong ErrCode = 10001
)

//通用代码
var (
	CodeSuccess                    = 0 // 成功
	CodeIllegalVoteListParamam     = 400001
	CodeSubscribeContractEventFail = 400002
	CodeIllegalIdParams            = 400003
	CodeUnmarshalFail              = 400004
	CodeSaveVoteDetailFail         = 400005
	CodeSaveVoteFail               = 400006
	CodeUpdateVoteDetailNumberFail = 400007
	CodeGetVoteListFail            = 400008
	CodeIllegalVoteDetailParamam   = 400009
	CodeGetVoteDetailFail          = 400010
	CodeGetVoteByUserIdFail        = 400011
	CodeNoAuth = 400012
	CodeTokenError = 400013
)

//通用错误
var (
	SuccessDealed              = "success"
	IllegalVoteListParamam     = "Illegal VoteListParamam"
	SubscribeContractEventFail = "SubscribeContractEvent Fail"
	IllegalIdParams            = "Illegal Id Params"
	UnmarshalFail              = "UnmarshalFail"
	SaveVoteDetailFail         = "Save VoteDetail Fail"
	SaveVoteFail               = "Save Vote Fail"
	UpdateVoteDetailNumberFail = "Update VoteDetailNumber Fail"
	GetVoteListFail            = "GetVoteList Fail"
	IllegalVoteDetailParamam   = "Illegal Vote Detail Paramam"
	GetVoteDetailFail          = "GetVoteDetail Fail"
	GetVoteByUserIdFail        = "GetVoteByUserId Fail"
	NoAuth = "No Auth"
	TokenError = "Token Error"
)

package tokenmiddleware

import (
	localutil "Dapp/local_util"
	"github.com/gin-gonic/gin"
	"net/http"
)

var (
	//Userinfokey = "user-info"
	TokenHeader = "x-token"
)
//
//func GenerateToken(user *models.UserToken) (string, error) {
//	expireTime := time.Now().Add(time.Duration(serverconfig.GlobalCFG.ApiCFG.TokenExpire) * time.Second)
//	claims := models.MyClaims{
//		User: *user,
//		RegisteredClaims: jwt.RegisteredClaims{
//			ExpiresAt: jwt.NewNumericDate(expireTime),
//			Issuer:    serverconfig.GlobalCFG.ApiCFG.JWTIssue,
//		},
//	}
//	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
//	if tokenStr, err := token.SignedString([]byte(serverconfig.GlobalCFG.ApiCFG.JWTSign)); err != nil {
//		return "", err
//	} else {
//		return tokenStr, nil
//	}
//}
//
//func ParseToken(tokenString string) (*models.MyClaims, error) {
//	claims := &models.MyClaims{}
//	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
//		return []byte(serverconfig.GlobalCFG.ApiCFG.JWTSign), nil
//	})
//	return claims, err
//}

func JWTAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenstr := ctx.Request.Header.Get(TokenHeader)
		if len(tokenstr) == 0 ||
			tokenstr != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJVVUlEIjoiMjNjMGIxOTktZGVjMS00ODMyLWIxOWQtOWJj" +
			"Yzg5ODQ3NTJmIiwiSUQiOjEsIlVzZXJuYW1lIjoiYWRtaW4iLCJOaWNrTmFtZSI6Iui2hee6p-euoeeQhuWRmCIsIlBob25lIjoiM" +
			"TU3MDEyMDg3MzYiLCJBdXRob3JpdHlJZCI6Ijg4OCIsIk9wZW5pZEFjY2Vzc1Rva2VuIjoiIiwiT3BlbmlkUmVmcmVzaFRva2VuIj" +
			"oiIiwiQnVmZmVyVGltZSI6ODY0MDAwLCJleHAiOjE2NjI2MjA1ODEsImlzcyI6ImNoYWlubWFrZXIiLCJuYmYiOjE2NTY1NzE1O" +
			"DF9.r5MNRuZkflcXou07GO0YSWB12VnFSfsTAloQos7iwsY"{
			ctx.Abort()
			ctx.JSONP(http.StatusOK, gin.H{
				"code": localutil.CodeNoAuth,
				"msg":  localutil.NoAuth,
			})
			return
		}
		//claim, claimErr := ParseToken(tokenstr)
		//if claimErr != nil {
		//	ctx.Abort()
		//	ctx.JSONP(http.StatusOK, gin.H{
		//		"code": localutil.CodeTokenError,
		//		"msg":  claimErr.Error(),
		//	})
		//	return
		//}
		//if claim.RegisteredClaims.ExpiresAt.Unix() < time.Now().Unix() {
		//	ctx.Abort()
		//	ctx.JSONP(http.StatusOK, gin.H{
		//		"code": localutil.CodeTokenExpire,
		//		"msg":  "token expired",
		//	})
		//	return
		//}
		//ctx.Set(Userinfokey, claim.User)
		ctx.Next()
	}
}

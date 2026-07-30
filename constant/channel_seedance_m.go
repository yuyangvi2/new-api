package constant

const ChannelTypeSeedanceM = 9006

func init() {
	registerCustomChannel(ChannelTypeSeedanceM, "Seedance-M", "https://zhenze-huhehaote.cmecloud.cn/api/v3")
}

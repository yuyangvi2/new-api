package constant

// ============================================================================
// 自定义渠道类型（fork 专属）
//
// 约定：自定义渠道一律使用 9000+ 高位号段，避免与上游顺序递增的渠道编号冲突。
// 上游短期内不可能用到 9000，故 git merge upstream 时既不会文本冲突、
// 更不会出现「上游的 58 和我们的 58 语义不同」导致 DB 里渠道类型被错认的问题。
//
// 所有注册都在本文件的 init() 里完成，不改动 channel.go 的字面量
// （常量块 / ChannelBaseURLs / ChannelTypeNames），把合并冲突面降到最低。
// ============================================================================

const (
	ChannelTypeVCLM  = 9001 // 腾讯云 VCLM 可灵图生视频（TC3 签名，任务型）
	ChannelTypeAIArt = 9002 // 腾讯云 AIART 大模型图像创作（Image-GI / Nano Banana，TC3 签名，任务型）
)

func init() {
	registerCustomChannel(ChannelTypeVCLM, "VCLM (Kling)", "https://vclm.tencentcloudapi.com")
	registerCustomChannel(ChannelTypeAIArt, "AIART (Image-GI)", "https://aiart.tencentcloudapi.com")
}

// registerCustomChannel 把高位自定义渠道安全地登记进上游的查表结构：
//   - 给 ChannelBaseURLs 这个「按下标访问的位置切片」补位，避免高位 type 越界 panic；
//   - 在 ChannelTypeNames 这个 map 里登记显示名。
func registerCustomChannel(channelType int, name, baseURL string) {
	for len(ChannelBaseURLs) <= channelType {
		ChannelBaseURLs = append(ChannelBaseURLs, "")
	}
	ChannelBaseURLs[channelType] = baseURL
	if ChannelTypeNames != nil {
		ChannelTypeNames[channelType] = name
	}
}

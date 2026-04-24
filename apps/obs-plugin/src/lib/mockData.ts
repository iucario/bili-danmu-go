import type { ChatItem, SuperChatEvent } from './types.js';

let _id = 0;
const id = () => String(++_id);
const ts = () => Math.floor(Date.now() / 1000);

export const mockItems: ChatItem[] = [
	{
		kind: 'text',
		data: {
			id: id(), timestamp: ts(), uid: '1',
			authorName: '普通观众', authorType: 0, privilegeType: 0,
			content: '主播好！今天也来了~',
			authorLevel: 12, isNewbie: false, isMobileVerified: true,
			isGiftDanmaku: false, isMirror: false,
			medalLevel: 0, medalName: '',
			avatarUrl: '', contentType: 0, contentTypeParams: {}, translation: '',
		},
	},
	{
		kind: 'text',
		data: {
			id: id(), timestamp: ts(), uid: '2',
			authorName: '守护粉丝', authorType: 0, privilegeType: 3,
			content: '加油！支持主播！',
			authorLevel: 25, isNewbie: false, isMobileVerified: true,
			isGiftDanmaku: false, isMirror: false,
			medalLevel: 12, medalName: '守护',
			avatarUrl: '', contentType: 0, contentTypeParams: {}, translation: '',
		},
	},
	{
		kind: 'text',
		data: {
			id: id(), timestamp: ts(), uid: '3',
			authorName: '提督大人', authorType: 0, privilegeType: 2,
			content: '这游戏真好看，继续！',
			authorLevel: 40, isNewbie: false, isMobileVerified: true,
			isGiftDanmaku: false, isMirror: false,
			medalLevel: 21, medalName: '大航海',
			avatarUrl: '', contentType: 0, contentTypeParams: {}, translation: '',
		},
	},
	{
		kind: 'text',
		data: {
			id: id(), timestamp: ts(), uid: '4',
			authorName: '房管酱', authorType: 2, privilegeType: 0,
			content: '请大家文明发言，谢谢～',
			authorLevel: 30, isNewbie: false, isMobileVerified: true,
			isGiftDanmaku: false, isMirror: false,
			medalLevel: 0, medalName: '',
			avatarUrl: '', contentType: 0, contentTypeParams: {}, translation: '',
		},
	},
	{
		kind: 'superchat',
		data: {
			id: id(), timestamp: ts(), uid: '5',
			authorName: '土豪观众', avatarUrl: '',
			price: 100, content: '主播辛苦了！每天都来看你直播，加油！',
			translation: '', privilegeType: 0, medalLevel: 0, medalName: '', time: 60,
		},
	},
	{
		kind: 'superchat',
		data: {
			id: id(), timestamp: ts(), uid: '5',
			authorName: '土豪观众', avatarUrl: '',
			price: 30, content: '主播辛苦了！每天都来看你直播，加油！',
			translation: '', privilegeType: 0, medalLevel: 0, medalName: '', time: 30,
		},
	},
	{
		kind: 'text',
		data: {
			id: id(), timestamp: ts(), uid: '6',
			authorName: '总督殿下', authorType: 0, privilegeType: 1,
			content: '哈哈哈这也太好笑了吧',
			authorLevel: 60, isNewbie: false, isMobileVerified: true,
			isGiftDanmaku: false, isMirror: false,
			medalLevel: 40, medalName: '总督团',
			avatarUrl: '', contentType: 0, contentTypeParams: {}, translation: '',
		},
	},
	{
		kind: 'text',
		data: {
			id: id(), timestamp: ts(), uid: '7',
			authorName: '新来的', authorType: 0, privilegeType: 0,
			content: '第一次来，主播好厉害！',
			authorLevel: 1, isNewbie: true, isMobileVerified: false,
			isGiftDanmaku: false, isMirror: false,
			medalLevel: 0, medalName: '',
			avatarUrl: '', contentType: 0, contentTypeParams: {}, translation: '',
		},
	},
	{
		kind: 'superchat',
		data: {
			id: id(), timestamp: ts(), uid: '8',
			authorName: '超级粉丝', avatarUrl: '',
			price: 500, content: '一直支持你！希望你越来越好！',
			translation: '', privilegeType: 3, medalLevel: 18, medalName: '铁粉', time: 120,
		},
	},
	{
		kind: 'text',
		data: {
			id: id(), timestamp: ts(), uid: '9',
			authorName: 'bot33', authorType: 0, privilegeType: 0,
			content: '这是机器人发的消息，可能是测试数据哦~ 这是机器人发的消息，可能是测试数据哦~',
			authorLevel: 1, isNewbie: true, isMobileVerified: false,
			isGiftDanmaku: false, isMirror: false,
			medalLevel: 0, medalName: '',
			avatarUrl: '', contentType: 0, contentTypeParams: {}, translation: '',
		},
	},
];

export const previewPinnedSCs = mockItems
	.filter((i) => i.kind === 'superchat')
	.map((i) => i.data) as SuperChatEvent[];

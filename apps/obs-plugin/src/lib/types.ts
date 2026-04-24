export interface TextEvent {
	id: string;
	timestamp: number;
	authorName: string;
	/** 0=normal, 1=guard, 2=admin, 3=room owner */
	authorType: number;
	content: string;
	/** 0=none, 1=总督, 2=提督, 3=舰长 */
	privilegeType: number;
	isGiftDanmaku: boolean;
	authorLevel: number;
	isNewbie: boolean;
	isMobileVerified: boolean;
	medalLevel: number;
	medalName: string;
	avatarUrl: string;
	uid: string;
	/** 0=text, 1=emoticon */
	contentType: number;
	contentTypeParams: Record<string, string>;
	isMirror: boolean;
	translation: string;
}

export interface SuperChatEvent {
	id: string;
	timestamp: number;
	authorName: string;
	avatarUrl: string;
	uid: string;
	/** Price in CNY */
	price: number;
	content: string;
	translation: string;
	privilegeType: number;
	medalLevel: number;
	medalName: string;
	/** Duration the SC should be pinned, in seconds */
	time: number;
}

export interface DelSuperChatEvent {
	ids: string[];
}

export type ChatItem =
	| { kind: 'text'; data: TextEvent }
	| { kind: 'superchat'; data: SuperChatEvent };

/** Returns Tailwind bg/text class pair for a SC price tier */
export function scColorClass(price: number): string {
	if (price >= 2000) return 'sc-red';
	if (price >= 1000) return 'sc-orange';
	if (price >= 500) return 'sc-yellow';
	if (price >= 200) return 'sc-green';
	if (price >= 100) return 'sc-teal';
	return 'sc-blue';
}

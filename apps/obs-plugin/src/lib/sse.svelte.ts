import type { ChatItem, DelSuperChatEvent, SuperChatEvent, TextEvent } from './types.js';

const MAX_CHAT_ITEMS = 50;

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected';

/**
 * Creates a reactive SSE store that connects to the danmu-go backend.
 * Uses Svelte 5 $state runes — must be called inside a Svelte component or effect root.
 */
export function createSSEStore(roomId: number) {
	let chatItems = $state<ChatItem[]>([]);
	let pinnedSCs = $state<SuperChatEvent[]>([]);
	let status = $state<ConnectionStatus>('connecting');
	let error = $state<string | null>(null);

	let es: EventSource | null = null;

	function connect() {
		if (es) es.close();
		status = 'connecting';

		const url = `/api/chat/stream?roomId=${roomId}`;
		es = new EventSource(url);

		es.addEventListener('open', () => {
			status = 'connected';
			error = null;
		});

		es.addEventListener('add_text', (e: MessageEvent) => {
			const ev = JSON.parse(e.data) as TextEvent;
			chatItems = [...chatItems.slice(-(MAX_CHAT_ITEMS - 1)), { kind: 'text', data: ev }];
		});

		es.addEventListener('add_super_chat', (e: MessageEvent) => {
			const ev = JSON.parse(e.data) as SuperChatEvent;
			chatItems = [...chatItems.slice(-(MAX_CHAT_ITEMS - 1)), { kind: 'superchat', data: ev }];
			pinnedSCs = [...pinnedSCs, ev];

			const duration = (ev.time ?? 60) * 1000;
			setTimeout(() => {
				pinnedSCs = pinnedSCs.filter((sc) => sc.id !== ev.id);
			}, duration);
		});

		es.addEventListener('del_super_chat', (e: MessageEvent) => {
			const ev = JSON.parse(e.data) as DelSuperChatEvent;
			pinnedSCs = pinnedSCs.filter((sc) => !ev.ids.includes(sc.id));
		});

		es.addEventListener('fatal_error', (e: MessageEvent) => {
			const ev = JSON.parse(e.data) as { type: string; msg: string };
			error = ev.msg;
			status = 'disconnected';
		});

		es.addEventListener('error', () => {
			// EventSource will auto-reconnect; reflect the transient disconnected state.
			status = 'disconnected';
		});
	}

	function disconnect() {
		es?.close();
		es = null;
		status = 'disconnected';
	}

	return {
		get chatItems() { return chatItems; },
		get pinnedSCs() { return pinnedSCs; },
		get status() { return status; },
		get error() { return error; },
		connect,
		disconnect
	};
}

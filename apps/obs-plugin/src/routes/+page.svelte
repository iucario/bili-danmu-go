<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { createSSEStore } from '$lib/sse.svelte.js';
	import ChatList from '$lib/ChatList.svelte';
	import SuperChatPin from '$lib/SuperChatPin.svelte';

	let roomId = $state<number | null>(null);
	let theme = $state('plain');
	let store = $state<ReturnType<typeof createSSEStore> | null>(null);

	const KNOWN_THEMES = ['plain', 'bubble', 'bubble-light'];

	onMount(() => {
		const params = new URLSearchParams(window.location.search);
		const t = params.get('theme') || '';
		theme = KNOWN_THEMES.includes(t) ? t : 'plain';
		const id = parseInt(params.get('roomId') ?? '', 10);
		if (!isNaN(id) && id > 0) {
			roomId = id;
			const filterLottery = params.get('filterLottery') === '1';
			store = createSSEStore(id, filterLottery);
			store.connect();
		}
	});

	onDestroy(() => store?.disconnect());
</script>

<div class="overlay theme-{theme}">
	{#if roomId === null}
		<div class="center-hint">
			Add <code>?roomId=12345</code> to the URL.
		</div>
	{:else if store}
		<SuperChatPin superchats={store.pinnedSCs} />
		<ChatList items={store.chatItems} />

		{#if store.status !== 'connected'}
			<div class="status-banner" class:disconnected={store.status === 'disconnected'}>
				{#if store.status === 'connecting'}
					<span class="dot pulse"></span> Connecting…
				{:else}
					<span class="dot"></span>
					{store.error ?? 'Disconnected — reconnecting…'}
				{/if}
			</div>
		{/if}
	{/if}
</div>

<style>
	.overlay {
		display: flex;
		flex-direction: column;
		width: 100vw;
		max-width: 800px;
		height: 100vh;
		background: transparent;
		overflow: hidden;
	}

	.center-hint {
		margin: auto;
		padding: 12px 20px;
		background: rgba(0, 0, 0, 0.6);
		color: #fff;
		border-radius: 8px;
		font-size: 14px;
	}

	.center-hint code {
		background: rgba(255, 255, 255, 0.15);
		padding: 2px 6px;
		border-radius: 4px;
	}

	/* Sticks to the bottom, only visible when not connected */
	.status-banner {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 5px 12px;
		font-size: 14px;
		background: rgba(30, 30, 30, 0.75);
		color: #ffffffcc;
	}

	.status-banner.disconnected {
		background: rgba(183, 28, 28, 0.8);
		color: #ffebee;
	}

	.dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: currentColor;
		flex-shrink: 0;
	}

	.dot.pulse {
		animation: pulse 1.2s ease-in-out infinite;
	}

	@keyframes pulse {
		0%,
		100% {
			opacity: 1;
		}
		50% {
			opacity: 0.25;
		}
	}
</style>

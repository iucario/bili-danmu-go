<script lang="ts">
	import type { SuperChatEvent } from './types.js';
	import { scColorClass, DEFAULT_AVATAR } from './types.js';

	let { superchats }: { superchats: SuperChatEvent[] } = $props();

	/** Track remaining seconds for each SC */
	let remaining = $state<Map<string, number>>(new Map());

	$effect(() => {
		// When a new SC arrives, start tracking its countdown
		for (const sc of superchats) {
			if (!remaining.has(sc.id)) {
				remaining = new Map(remaining).set(sc.id, sc.time ?? 60);
			}
		}
	});

	// Tick countdown every second
	$effect(() => {
		const interval = setInterval(() => {
			if (remaining.size === 0) return;
			const next = new Map<string, number>();
			for (const [id, secs] of remaining) {
				if (secs > 1) next.set(id, secs - 1);
				// when secs reaches 0 we omit it — sse.svelte.ts handles removal from superchats
			}
			remaining = next;
		}, 1000);
		return () => clearInterval(interval);
	});

	function formatTime(secs: number): string {
		const m = Math.floor(secs / 60);
		const s = secs % 60;
		return m > 0 ? `${m}:${String(s).padStart(2, '0')}` : `${s}s`;
	}

	const MAX_SC_CARDS = 2;
	let visibleChats = $derived(superchats.slice(0, MAX_SC_CARDS));
</script>

{#if superchats.length > 0}
	<div class="pinned-zone">
		{#each visibleChats as sc (sc.id)}
			<div class="sc-card {scColorClass(sc.price)}">
				<div class="sc-header">
					<img class="avatar" src={sc.avatarUrl || DEFAULT_AVATAR} alt={sc.authorName}
					onerror={(e) => { (e.currentTarget as HTMLImageElement).src = DEFAULT_AVATAR; }} />
					<div class="sc-meta">
						<span class="sc-author">{sc.authorName}</span>
						{#if sc.medalName && sc.medalLevel}
							<span class="sc-medal">{sc.medalName} {sc.medalLevel}</span>
						{/if}
					</div>
					<div class="sc-right">
						<span class="sc-price">¥{sc.price}</span>
						{#if remaining.has(sc.id)}
							<span class="sc-timer">{formatTime(remaining.get(sc.id)!)}</span>
						{/if}
					</div>
				</div>
				{#if sc.content}
					<div class="sc-body">{sc.content}</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}

<style>
	.pinned-zone {
		display: flex;
		flex-direction: column;
		gap: calc(4px * var(--scale, 1));
		padding: calc(6px * var(--scale, 1)) calc(8px * var(--scale, 1)) calc(2px * var(--scale, 1));
		max-height: 45%;
		overflow: hidden; /* safety: JS already limits to fully-fitting cards */
		flex-shrink: 0;
	}

	/* SC card layout styles are in layout.css (shared with ChatList) */
</style>

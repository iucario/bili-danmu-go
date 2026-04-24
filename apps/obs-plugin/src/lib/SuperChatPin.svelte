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
</script>

{#if superchats.length > 0}
	<div class="pinned-zone">
		{#each superchats as sc (sc.id)}
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
					<div class="sc-content">{sc.content}</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}

<style>
	.pinned-zone {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 6px 8px 2px;
	}

	.sc-card {
		border-radius: 8px;
		overflow: hidden;
		font-size: 14px;
	}

	.sc-header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 10px;
	}

	.avatar {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		object-fit: cover;
		flex-shrink: 0;
	}

	.sc-meta {
		flex: 1;
		display: flex;
		flex-direction: column;
		min-width: 0;
	}

	.sc-author {
		font-weight: 700;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.sc-medal {
		font-size: 11px;
		opacity: 0.8;
	}

	.sc-right {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		flex-shrink: 0;
	}

	.sc-price {
		font-weight: 700;
		font-size: 15px;
	}

	.sc-timer {
		font-size: 11px;
		opacity: 0.75;
		font-variant-numeric: tabular-nums;
	}

	.sc-content {
		padding: 6px 10px 8px;
		border-top: 1px solid rgba(255, 255, 255, 0.15);
		line-height: 1.4;
		word-break: break-word;
	}

	/* Color themes live in layout.css — .sc-card reads --sc-bg / --sc-color from the tier class */
	.sc-card {
		background: var(--sc-bg, rgba(13, 71, 161, 0.88));
		color: var(--sc-color, #e3f2fd);
	}
</style>

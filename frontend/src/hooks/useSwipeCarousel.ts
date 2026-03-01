import { useCallback, useEffect, useRef, useState, type RefObject } from "react";

type SwipeCarouselOptions = {
  count: number;
  settleDelayMs?: number;
  swipeThresholdPx?: number;
  snapThresholdRatio?: number;
};

type SwipeCarouselResult = {
  index: number;
  isInteracting: boolean;
  viewportHeight: number | null;
  trackRef: RefObject<HTMLDivElement | null>;
  setIndex: (nextIndex: number) => void;
  scrollToIndex: (nextIndex: number, behavior: ScrollBehavior) => void;
  refreshHeight: () => void;
  onTrackScroll: () => void;
  onPointerDown: (clientX: number) => void;
  onPointerUp: (clientX: number) => void;
  onPointerCancel: () => void;
};

export function useSwipeCarousel({
  count,
  settleDelayMs = 140,
  swipeThresholdPx = 18,
  snapThresholdRatio = 0.35,
}: SwipeCarouselOptions): SwipeCarouselResult {
  const [index, setIndexState] = useState(0);
  const [viewportHeight, setViewportHeight] = useState<number | null>(null);
  const [isInteracting, setIsInteracting] = useState(false);
  const trackRef = useRef<HTMLDivElement | null>(null);
  const settleTimeoutRef = useRef<number | null>(null);
  const dragStartXRef = useRef<number | null>(null);
  const dragStartIndexRef = useRef<number | null>(null);
  const gestureTargetIndexRef = useRef<number | null>(null);

  const clampIndex = useCallback(
    (value: number) => Math.max(0, Math.min(count - 1, value)),
    [count],
  );

  const getNearestIndex = useCallback(() => {
    const track = trackRef.current;
    if (!track || count === 0) {
      return 0;
    }
    const width = track.clientWidth;
    if (width <= 0) {
      return 0;
    }
    const rawIndex = track.scrollLeft / width;
    const lowerIndex = Math.floor(rawIndex);
    const progressToNext = rawIndex - lowerIndex;
    const snapIndex =
      progressToNext >= snapThresholdRatio ? lowerIndex + 1 : lowerIndex;
    return clampIndex(snapIndex);
  }, [clampIndex, count, snapThresholdRatio]);

  const syncViewportHeight = useCallback((targetIndex: number) => {
    const track = trackRef.current;
    if (!track) {
      return;
    }
    const activeCard = track.children.item(targetIndex) as HTMLElement | null;
    if (!activeCard) {
      return;
    }
    setViewportHeight(activeCard.offsetHeight);
  }, []);

  const scrollToIndex = useCallback(
    (nextIndex: number, behavior: ScrollBehavior) => {
      const track = trackRef.current;
      if (!track) {
        return;
      }
      const width = track.clientWidth;
      if (width <= 0) {
        return;
      }
      track.scrollTo({ left: clampIndex(nextIndex) * width, behavior });
    },
    [clampIndex],
  );

  const setIndex = useCallback(
    (nextIndex: number) => {
      setIndexState(clampIndex(nextIndex));
    },
    [clampIndex],
  );

  const refreshHeight = useCallback(() => {
    syncViewportHeight(index);
  }, [index, syncViewportHeight]);

  const settle = useCallback(() => {
    const targetIndex = gestureTargetIndexRef.current ?? getNearestIndex();
    gestureTargetIndexRef.current = null;
    setIndexState(targetIndex);
    scrollToIndex(targetIndex, "auto");
    setIsInteracting(false);
    syncViewportHeight(targetIndex);
  }, [getNearestIndex, scrollToIndex, syncViewportHeight]);

  const onTrackScroll = useCallback(() => {
    const lockedTargetIndex = gestureTargetIndexRef.current;
    if (lockedTargetIndex != null) {
      setIndexState((prev) => (prev === lockedTargetIndex ? prev : lockedTargetIndex));
      setIsInteracting(true);
      if (settleTimeoutRef.current != null) {
        window.clearTimeout(settleTimeoutRef.current);
      }
      settleTimeoutRef.current = window.setTimeout(settle, settleDelayMs);
      return;
    }

    const nextIndex = getNearestIndex();
    setIndexState((prev) => (prev === nextIndex ? prev : nextIndex));
    setIsInteracting(true);
    if (settleTimeoutRef.current != null) {
      window.clearTimeout(settleTimeoutRef.current);
    }
    settleTimeoutRef.current = window.setTimeout(settle, settleDelayMs);
  }, [getNearestIndex, settle, settleDelayMs]);

  const onGestureStart = useCallback((clientX: number) => {
    dragStartXRef.current = clientX;
    dragStartIndexRef.current = getNearestIndex();
    setIsInteracting(true);
  }, [getNearestIndex]);

  const onGestureEnd = useCallback(
    (clientX: number) => {
      const dragStartX = dragStartXRef.current;
      const dragStartIndex = dragStartIndexRef.current;
      dragStartXRef.current = null;
      dragStartIndexRef.current = null;
      if (dragStartX == null) {
        return;
      }
      const deltaX = dragStartX - clientX;
      const trackWidth = trackRef.current?.clientWidth ?? 0;
      const relativeThresholdPx = trackWidth > 0 ? Math.round(trackWidth * 0.08) : 0;
      const effectiveThresholdPx = Math.max(swipeThresholdPx, relativeThresholdPx);
      const hasSwipeIntent = Math.abs(deltaX) >= effectiveThresholdPx;
      const baseIndex = dragStartIndex ?? getNearestIndex();
      const nextIndex = hasSwipeIntent
        ? clampIndex(baseIndex + (deltaX > 0 ? 1 : -1))
        : getNearestIndex();
      gestureTargetIndexRef.current = hasSwipeIntent ? nextIndex : null;
      if (settleTimeoutRef.current != null) {
        window.clearTimeout(settleTimeoutRef.current);
        settleTimeoutRef.current = null;
      }
      setIndexState(nextIndex);
      scrollToIndex(nextIndex, "smooth");
      syncViewportHeight(nextIndex);
      setIsInteracting(false);
    },
    [clampIndex, getNearestIndex, scrollToIndex, swipeThresholdPx, syncViewportHeight],
  );

  useEffect(() => {
    if (count === 0) {
      setIndexState(0);
      setViewportHeight(null);
      return;
    }
    let rafId = 0;
    setIndexState((prev) => {
      const targetIndex = clampIndex(prev);
      rafId = window.requestAnimationFrame(() => {
        scrollToIndex(targetIndex, "auto");
        syncViewportHeight(targetIndex);
      });
      return targetIndex;
    });
    return () => window.cancelAnimationFrame(rafId);
  }, [clampIndex, count, scrollToIndex, syncViewportHeight]);

  useEffect(() => {
    return () => {
      if (settleTimeoutRef.current != null) {
        window.clearTimeout(settleTimeoutRef.current);
      }
    };
  }, []);

  return {
    index,
    isInteracting,
    viewportHeight,
    trackRef,
    setIndex,
    scrollToIndex,
    refreshHeight,
    onTrackScroll,
    onPointerDown: onGestureStart,
    onPointerUp: onGestureEnd,
    onPointerCancel: () => {
      dragStartXRef.current = null;
      dragStartIndexRef.current = null;
      gestureTargetIndexRef.current = null;
      setIsInteracting(false);
    },
  };
}

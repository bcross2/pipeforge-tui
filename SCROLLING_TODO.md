## Scrolling Issue — PipeForge TUI

### Problem
When the pipeline has more blocks than fit in the canvas panel, scrolling
does not follow the cursor reliably. The bottom blocks get clipped by the
panel box and the user cannot see them.

### Root Cause
There is a mismatch between how many lines the scroll logic THINKS are
visible vs how many lines the panelBox ACTUALLY displays.

The render flow has two layers that both constrain height:

    RenderCanvas(height)         ← scroll logic uses this to decide
         │                          how many lines to show
         ▼
    panelBox(content, height)    ← lipgloss clips content to
                                    Height(height-2) and MaxHeight(height)

The scroll logic in RenderCanvas calculates:

    visibleLines = height - 6

But the actual visible lines depend on:
  - panelBox border:     2 lines (top + bottom)
  - panelBox MaxHeight:  hard clips anything beyond
  - title line:          1 line
  - blank/indicator:     1 line
  - scroll indicator:    1 line (if showing "...N below")

So the real available lines for block entries is:

    actual = (height - 2 border) - 1 title - 1 indicator top - 1 indicator bottom
           = height - 5

But this can vary because:
  - The "...N above" line only shows when scrollOffset > 0
  - The "...N below" line only shows when end < len(entries)
  - The command "$ ..." line at the bottom takes 2 entries (blank + cmd)
  - Each block takes 1 line, each connector takes 1 line
  - The file name header takes 2 lines (name + connector)

### What Needs To Change

1. SINGLE SOURCE OF TRUTH for visible line count
   Right now both RenderCanvas and panelBox independently constrain height.
   Either:
   (a) Remove Height/MaxHeight from panelBox and let the renderer handle
       all clipping (simpler, but risks overflow if math is wrong)
   (b) Have RenderCanvas return exactly the right number of lines and
       panelBox just wraps it in a border (preferred)

2. SEPARATE THE COMMAND BAR FROM THE SCROLLABLE AREA
   The "$ grep ... | sort ..." line at the bottom is part of the entries
   list and gets scrolled away. It should be pinned at the bottom of the
   canvas panel, outside the scrollable block list.

3. CONSISTENT SCROLL FORMULA
   For the scrollable block list:
     totalEntries = len(blockEntries)  // just blocks + connectors
     visibleSlots = panelInnerHeight - titleLines - cmdBarLines
     scrollOffset = clamp so cursor stays within [offset, offset+visibleSlots)

4. TEST WITH EDGE CASES
   - 0 blocks (empty state)
   - 1 block (no scrolling needed)
   - Exactly N blocks where N fills the panel (boundary)
   - N+1 blocks (first scroll trigger)
   - 10+ blocks (deep scrolling)
   - Very short terminal (< 20 rows)

### Same Issue Exists In
- Library panel (same pattern, same potential mismatch)
- Preview panel (uses maxRows param, may also be off)

### Quick Fix Approach
Pass the INNER height (height - 2 for borders) from layout.go into the
render functions instead of the outer height. Then the renderers only need
to subtract their own overhead (title, indicators) without guessing about
borders.

In layout.go change:
    canvas := RenderCanvas(..., canvasHeight, ...)
To:
    canvas := RenderCanvas(..., canvasHeight - 2, ...)  // pass inner height

Then in canvas.go:
    visibleLines = height - 3  // title + top indicator + bottom indicator

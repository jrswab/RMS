# Plan: Fix Ingredient Row Focus Bug

## What is being built and why?
We are fixing a bug where input fields in the `ingredient-row` lose focus after every character entered. 

**Root Cause:**
The `data-effect` that renders the ingredient rows is being triggered on every keystroke. The current implementation uses `window.renderIngredientRows(@peek(() => $ingredients))`. 
While `@peek` correctly disables dependency tracking while evaluating the arrow function, it returns the deep reactive Proxy object for `$ingredients`. When `window.renderIngredientRows` iterates over this Proxy *outside* of the `@peek` callback, Datastar's dependency tracking has already been re-enabled. This registers dependencies on the nested properties (like `ingredients.0.foodName`). 
When `data-bind` updates these properties on keystroke, the effect is triggered, replacing `innerHTML` and causing the input to be destroyed and lose focus.

**Solution:**
Move the `window.renderIngredientRows` call *inside* the `@peek` callback so that all property accesses happen while dependency tracking is disabled.

## What are the exact boundaries (what is NOT included)?
- We are only fixing the `data-effect` attribute in `views/pages/recipe_form.templ`.
- We are NOT rewriting the client-side rendering logic to server-side rendering (which is generally preferred in Datastar/HTMX), as that would be a larger refactor outside the scope of this specific bug fix.

## What are the discrete tasks @build should delegate to subagents?
1. **Edit `views/pages/recipe_form.templ`**:
   - Locate the `data-effect` attribute on the `#ingredient-rows` div (around line 60).
   - Change it from:
     `data-effect="$_ingredientRowsVersion; el.innerHTML = window.renderIngredientRows(@peek(() => $ingredients))"`
   - To:
     `data-effect="$_ingredientRowsVersion; el.innerHTML = @peek(() => window.renderIngredientRows($ingredients))"`
2. **Run `templ generate`**:
   - Execute `templ generate` in the project root to update the generated Go code.

## What does done look like — how will @qa know it passed?
- The `data-effect` attribute in `views/pages/recipe_form.templ` is updated as specified.
- `templ generate` runs successfully without errors.
- When a user types in the ingredient input fields, the fields do not lose focus, allowing for continuous typing.

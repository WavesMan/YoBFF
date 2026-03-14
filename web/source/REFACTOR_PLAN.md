# Frontend Refactoring Plan

Based on the `modern-ui-demo.html` design and the current codebase structure, here is the plan to refactor the YoBFF frontend.

## 1. Design System Foundation
**Goal**: Establish a consistent styling foundation based on the demo.

- **Global CSS (`src/index.css`)**:
  - Replace existing variables with the comprehensive set from `modern-ui-demo.html` (Colors, Shadows, Radius, Transitions).
  - Add global reset and typography styles.
  - Remove unused variables.

## 2. Component Library (Atomic Design)
**Goal**: Create reusable, decoupled UI components in `src/components/ui`.

- [x] **Button**: `Button.tsx` (Variants: primary, secondary, danger, ghost; Sizes: sm, md, lg).
- [x] **Input/Form**: `Input.tsx`, `Select.tsx` (Standardized styles with focus states).
- [x] **Layout**: `Card.tsx` (Container for stats and details), `Badge.tsx` (Status indicators).
- [ ] **Navigation**: `Tabs.tsx` (Controlled tab switching with animations).
- [ ] **Feedback**: `Toast.tsx` (Notification system), `Modal.tsx` (Dialogs), `Drawer.tsx` (Slide-over panels).
- [x] **Data Display**: `Table.tsx` (Composable table components).

Note: `Modal.tsx` and `Drawer.tsx` have been implemented but `Toast.tsx` is pending.

## 3. Layout Refactoring
**Goal**: Align the application shell with the demo's layout.

- **Sidebar (`src/components/Sidebar.tsx`)**:
  - Update HTML structure to match the demo.
  - Use new CSS variables and classes.
  - Implement collapsible logic cleaner.
- **TopBar (`src/components/TopBar.tsx`)**:
  - Add Breadcrumbs support.
  - Add Search Input.
  - Add User/Action buttons.

## 4. Feature Refactoring
**Goal**: Update existing views to use the new components and reduce file size.

- **Site List (`src/sections/traffic/SiteList.tsx`)**:
  - Replace HTML tables with `Table` component.
  - Use `Badge` for status.
  - Use `Button` for actions.
  - Implement "Dropdown Menu" for row actions as seen in demo.

- **Site Config Drawer (`src/sections/traffic/site-config/SiteConfigDrawer.tsx`)**:
  - **Problem**: Current file size > 600 lines.
  - **Solution**:
    - Extract state management into a custom hook `useSiteConfig`.
    - Use the new `Drawer` and `Tabs` components.
    - Delegate API calls to `src/admin/api.ts` (already done) but wrap in hooks for cleaner component logic.

## 5. Implementation Steps
1.  **Setup**: Update `index.css` and create `src/components/ui`.
2.  **Components**: Implement core UI components one by one.
3.  **Layout**: Refactor Sidebar and TopBar.
4.  **Views**: Refactor SiteList and SiteConfigDrawer.
5.  **Cleanup**: Remove unused CSS and legacy code.

## 6. Constraints Checklist
- [ ] Single file lines < 600.
- [ ] No non-style logic changes.
- [ ] Component-based architecture.
- [ ] Low coupling.

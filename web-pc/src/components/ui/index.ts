/**
 * HiGoOS in-house UI component library.
 *
 * Import named components via a relative path, e.g.
 *   import { UiButton, UiModal } from '../../ui';
 * Components are tree-shakeable — import only what each window uses.
 */

export { default as UiButton } from './Button/Button.vue';
export { default as UiIconButton } from './IconButton/IconButton.vue';
export { default as UiModal } from './Modal/Modal.vue';
export { default as UiConfirmDialog } from './ConfirmDialog/ConfirmDialog.vue';
export { default as UiSpinner } from './Spinner/Spinner.vue';
export { default as UiLoadingOverlay } from './Spinner/LoadingOverlay.vue';
export { default as UiEmptyState } from './EmptyState/EmptyState.vue';
export { default as UiCard } from './Card/Card.vue';
export { default as UiPanel } from './Card/Panel.vue';
export { default as UiBadge } from './Badge/Badge.vue';
export { default as UiTag } from './Badge/Tag.vue';
export { default as UiProgressBar } from './ProgressBar/ProgressBar.vue';
export { default as UiInput } from './Input/Input.vue';
export { default as UiTextarea } from './Input/Textarea.vue';
export { default as UiSelect } from './Select/Select.vue';
export { default as UiFormField } from './FormField/FormField.vue';
export { default as UiSwitch } from './Switch/Switch.vue';
export { default as UiCheckbox } from './Checkbox/Checkbox.vue';
export { default as UiRadio } from './Radio/Radio.vue';
export { default as UiSlider } from './Slider/Slider.vue';
export { default as UiSegmented } from './Segmented/Segmented.vue';
export { default as UiTabs } from './Tabs/Tabs.vue';
export { default as UiTabPanel } from './Tabs/TabPanel.vue';
export { default as UiTooltip } from './Tooltip/Tooltip.vue';
export { default as UiDropdown } from './Menu/Dropdown.vue';
export { default as UiMenuItem } from './Menu/MenuItem.vue';
export { default as UiDataTable } from './DataTable/DataTable.vue';
export { default as UiToastHost } from './Toast/ToastHost.vue';
export { default as UiConfirmHost } from './ConfirmDialog/ConfirmHost.vue';

export { useToast } from './Toast/useToast';
export { useConfirm } from './ConfirmDialog/useConfirm';

export type { UiTone, UiSize } from './tokens';
export type { SelectOption } from './Select/Select.vue';
export type { SegmentedOption } from './Segmented/Segmented.vue';
export type { TabItem } from './Tabs/Tabs.vue';
export type { Column, SortState } from './DataTable/DataTable.vue';
export type { ToastOptions } from './Toast/useToast';
export type { ConfirmOptions } from './ConfirmDialog/useConfirm';

// useNoticeDelete — coordinates confirm-dialog state for a delete action
import { useConfirmDialog } from './useConfirmDialog.ts';

export function useNoticeDelete(deleteNotice: (seq: number) => void) {
  const dialog = useConfirmDialog<number>();

  const handleConfirm = () => {
    const seq = dialog.confirm();
    if (seq != null) deleteNotice(seq);
  };

  return {
    dialogOpen: dialog.isOpen,
    openDialog: dialog.open,
    closeDialog: dialog.close,
    handleConfirm,
  };
}

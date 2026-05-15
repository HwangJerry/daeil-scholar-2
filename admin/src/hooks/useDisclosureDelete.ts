// useDisclosureDelete — coordinates confirm-dialog state for a disclosure delete action
import { useConfirmDialog } from './useConfirmDialog.ts';

export function useDisclosureDelete(deleteDisclosure: (seq: number) => void) {
  const dialog = useConfirmDialog<number>();

  const handleConfirm = () => {
    const seq = dialog.confirm();
    if (seq != null) deleteDisclosure(seq);
  };

  return {
    dialogOpen: dialog.isOpen,
    openDialog: dialog.open,
    closeDialog: dialog.close,
    handleConfirm,
  };
}

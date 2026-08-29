import { useId, useMemo, useState, type ChangeEvent, type CSSProperties } from 'react';
import {
  DndContext,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core';
import {
  SortableContext,
  arrayMove,
  rectSortingStrategy,
  sortableKeyboardCoordinates,
  useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { GripVertical, ImagePlus, X } from 'lucide-react';
import { uploadImage } from '../editor/uploadImage.ts';
import { Button } from '../ui/Button.tsx';
import { useToast } from '../../hooks/useToast.ts';

interface BannerImageManagerProps {
  imageUrls: string[];
  onChange: (urls: string[]) => void;
}

interface SortableImageItem {
  id: string;
  imageUrl: string;
}

function buildSortableItems(imageUrls: string[]): SortableImageItem[] {
  return imageUrls.map((imageUrl, index) => ({
    id: `${index}::${imageUrl}`,
    imageUrl,
  }));
}

export function BannerImageManager({ imageUrls, onChange }: BannerImageManagerProps) {
  const inputId = useId();
  const addToast = useToast((state) => state.addToast);
  const [isUploading, setIsUploading] = useState(false);

  const sortableItems = useMemo(() => buildSortableItems(imageUrls), [imageUrls]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const handleFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    event.target.value = '';
    if (!files.length) return;

    setIsUploading(true);

    const uploadedUrls: string[] = [];
    for (const file of files) {
      try {
        const url = await uploadImage(file, 'bannerAd');
        uploadedUrls.push(url);
      } catch {
        addToast({
          variant: 'error',
          title: '이미지 업로드 실패',
          description: `${file.name} 업로드 중 오류가 발생했습니다.`,
        });
      }
    }

    if (uploadedUrls.length > 0) {
      onChange([...imageUrls, ...uploadedUrls]);
    }

    setIsUploading(false);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = sortableItems.findIndex((item) => item.id === active.id);
    const newIndex = sortableItems.findIndex((item) => item.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;

    onChange(arrayMove(imageUrls, oldIndex, newIndex));
  };

  const handleRemove = (targetIndex: number) => {
    onChange(imageUrls.filter((_, index) => index !== targetIndex));
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-cool-gray">드래그로 대표 순서와 노출 순서를 조정할 수 있습니다.</p>
        <div>
          <input
            id={inputId}
            type="file"
            multiple
            accept="image/*"
            className="hidden"
            onChange={(event) => { void handleFileChange(event); }}
          />
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              const input = document.getElementById(inputId) as HTMLInputElement | null;
              input?.click();
            }}
            disabled={isUploading}
          >
            <ImagePlus className="mr-1 h-4 w-4" />
            {isUploading ? '업로드 중...' : '이미지 추가'}
          </Button>
        </div>
      </div>

      {sortableItems.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-border-light bg-background px-4 py-10 text-center text-sm text-cool-gray">
          사진을 등록해주세요.
        </div>
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={sortableItems.map((item) => item.id)} strategy={rectSortingStrategy}>
            <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
              {sortableItems.map((item, index) => (
                <BannerImageCard
                  key={item.id}
                  id={item.id}
                  imageUrl={item.imageUrl}
                  index={index}
                  onRemove={() => handleRemove(index)}
                  disabled={isUploading}
                />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}
    </div>
  );
}

interface BannerImageCardProps {
  id: string;
  imageUrl: string;
  index: number;
  onRemove: () => void;
  disabled: boolean;
}

function BannerImageCard({ id, imageUrl, index, onRemove, disabled }: BannerImageCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    disabled,
  });

  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.55 : 1,
    boxShadow: isDragging ? '0 12px 28px rgba(15, 23, 42, 0.18)' : undefined,
    zIndex: isDragging ? 10 : undefined,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="overflow-hidden rounded-2xl border border-border-light bg-white shadow-sm"
    >
      <div className="relative aspect-[4/3] bg-background">
        <img src={imageUrl} alt={`배너 이미지 ${index + 1}`} className="h-full w-full object-cover" />
        <button
          type="button"
          onClick={onRemove}
          disabled={disabled}
          className="absolute right-2 top-2 inline-flex h-8 w-8 items-center justify-center rounded-full bg-black/65 text-white transition hover:bg-black/80 disabled:opacity-50"
          aria-label="이미지 삭제"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
      <div className="flex items-center justify-between gap-2 px-3 py-2">
        <span className="text-xs font-medium text-dark-slate">이미지 {index + 1}</span>
        <button
          type="button"
          aria-label="이미지 순서 변경"
          className={`inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border-light text-cool-gray ${
            disabled ? 'pointer-events-none opacity-30' : 'cursor-grab hover:bg-background active:cursor-grabbing'
          }`}
          {...attributes}
          {...listeners}
          disabled={disabled}
        >
          <GripVertical className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

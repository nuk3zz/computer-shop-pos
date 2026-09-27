import { useEffect, useMemo, useRef, useState } from 'react'
import { ChevronLeft, ChevronRight, X } from 'lucide-react'
import { getMediaUrl } from '@/lib/media'
import type { Product } from '@/types'

interface ProductImageGalleryProps {
  product: Product | null
  onClose: () => void
}

export function ProductImageGallery({ product, onClose }: ProductImageGalleryProps) {
  const images = useMemo(() => product ? (product.images?.length ? product.images : product.image_url ? [product.image_url] : []) : [], [product])
  const [activeIndex, setActiveIndex] = useState(0)
  const touchStartX = useRef<number | null>(null)

  const previous = () => setActiveIndex((index) => (index - 1 + images.length) % images.length)
  const next = () => setActiveIndex((index) => (index + 1) % images.length)

  useEffect(() => setActiveIndex(0), [product])

  useEffect(() => {
    if (!product) return
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
      if (event.key === 'ArrowLeft' && images.length > 1) previous()
      if (event.key === 'ArrowRight' && images.length > 1) next()
    }
    window.addEventListener('keydown', handleKey)
    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener('keydown', handleKey)
    }
  }, [images.length, onClose, product])

  if (!product || images.length === 0) return null

  return (
    <div className="fixed inset-0 z-[100] flex flex-col bg-black/95 text-white" role="dialog" aria-modal="true" aria-label={`${product.name} photo gallery`} onClick={onClose}>
      <div className="flex items-center justify-between gap-4 px-4 py-3 sm:px-6">
        <div className="min-w-0">
          <h2 className="truncate font-semibold">{product.name}</h2>
          <p className="text-xs text-white/65">Photo {activeIndex + 1} of {images.length}</p>
        </div>
        <button type="button" onClick={onClose} className="rounded-full bg-white/10 p-2 hover:bg-white/20" aria-label="Close photo gallery"><X className="h-6 w-6" /></button>
      </div>

      <div
        className="relative flex min-h-0 flex-1 items-center justify-center px-3 pb-3 sm:px-16"
        onClick={(event) => event.stopPropagation()}
        onTouchStart={(event) => { touchStartX.current = event.changedTouches[0].clientX }}
        onTouchEnd={(event) => {
          if (touchStartX.current === null || images.length < 2) return
          const distance = event.changedTouches[0].clientX - touchStartX.current
          if (Math.abs(distance) > 45) distance > 0 ? previous() : next()
          touchStartX.current = null
        }}
      >
        <img src={getMediaUrl(images[activeIndex])} alt={`${product.name} photo ${activeIndex + 1}`} className="max-h-full max-w-full select-none object-contain" draggable={false} />
        {images.length > 1 && (
          <>
            <button type="button" onClick={previous} className="absolute left-3 rounded-full bg-black/50 p-2 hover:bg-white/20 sm:left-6" aria-label="Previous photo"><ChevronLeft className="h-7 w-7" /></button>
            <button type="button" onClick={next} className="absolute right-3 rounded-full bg-black/50 p-2 hover:bg-white/20 sm:right-6" aria-label="Next photo"><ChevronRight className="h-7 w-7" /></button>
          </>
        )}
      </div>

      {images.length > 1 && (
        <div className="flex justify-center gap-2 overflow-x-auto px-4 py-3" onClick={(event) => event.stopPropagation()}>
          {images.map((image, index) => (
            <button key={`${image}-${index}`} type="button" onClick={() => setActiveIndex(index)} className={`h-12 w-12 shrink-0 overflow-hidden rounded border-2 ${index === activeIndex ? 'border-white' : 'border-transparent opacity-60'}`} aria-label={`View photo ${index + 1}`}>
              <img src={getMediaUrl(image)} alt="" className="h-full w-full object-cover" />
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

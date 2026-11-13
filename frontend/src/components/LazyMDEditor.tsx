import { lazy, Suspense, type ComponentProps } from 'react'
import { Skeleton } from '@/components/ui/skeleton'

const MDEditor = lazy(() => import('@uiw/react-md-editor'))

type LazyMDEditorProps = ComponentProps<typeof MDEditor> & {
  fallbackHeight: number
}

export function LazyMDEditor({ fallbackHeight, ...props }: LazyMDEditorProps) {
  return (
    <Suspense
      fallback={<Skeleton className="w-full" style={{ height: fallbackHeight }} />}
    >
      <MDEditor {...props} />
    </Suspense>
  )
}

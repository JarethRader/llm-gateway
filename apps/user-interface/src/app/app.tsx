import { ThemeProvider } from "@/components/theme-provider"
import { Toaster } from "@/components/ui/sonner";
import { BackendProvider } from "@/store/backend";
import { ModelsList } from "./models-list";

export function App() {
  return (
    <ThemeProvider defaultTheme="dark">
      <BackendProvider>
        <div className="mt-8 w-full place-content-start justify-items-center-safe">
          <div className="w-3/4">
            <ModelsList />
          </div>
        </div>
        <Toaster />
      </BackendProvider>
    </ThemeProvider>
  );
}

export default App;

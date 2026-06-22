import React from "react";
import { Plus, Edit } from "lucide-react";
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { StatusPill } from "@/components/ui/status-pill";
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { ModelForm } from "@/components/forms/model-form";
import { useBackend } from "@/store/backend";
import { GetBackend, GetBackendList } from "@/api/backend";
import { toast } from "sonner";

export function ModelsList() {
  const { state, dispatch } = useBackend();

  const [showModal, setShowModal] = React.useState(false);
  const toggleModal = () => setShowModal(!showModal);

  const [backends, setBackend] = React.useState<SparseBackend[]>([]);

  React.useEffect(() => {
    if (state.BackendList.length) {
      setBackend(state.BackendList);
    }
  }, [state]);

  React.useEffect(() => {
    const hydrateBackends = async () => {
      try {
        const backendList = await GetBackendList();
        dispatch({
          type: "SET_BACKEND_LIST",
          payload: backendList,
        });
      } catch (err) {
        toast.error((err as Error).message, { position: "top-center" });
      }
    }

    hydrateBackends();
  }, [dispatch]);

  const handleSelectBackend = async (id: number) => {
    try {
      const selectedBackend = await GetBackend(id);
      dispatch({
        type: "SET_CURRENT_BACKEND",
        payload: selectedBackend,
      });
      setShowModal(true);
    } catch (err) {
      toast.error((err as Error).message, { position: "top-center" });
    }
  }

  return (
    <div>
      {showModal &&
        <div className="fixed inset-0 z-10 overflow-y-auto overscroll-contain backdrop-blur-sm">
          <div className="flex min-h-full items-center justify-center px-4 py-10">
            <ModelForm close={toggleModal} />
          </div>
        </div>
      }
      <div className="flex justify-between">
        <h1 className="scroll-m-20 text-left text-4xl font-semibold tracking-tight text-balance font-heading">Backends</h1>
        <Button className="cursor-pointer" size="lg" onClick={toggleModal}>
          <Plus />
          Add Backend
        </Button>
      </div>
      <Table className="table-fixed">
        <TableHeader>
          <TableRow>
            <TableHead className="w-[22%]">Backend</TableHead>
            <TableHead className="w-[33%]">Models</TableHead>
            <TableHead className="w-[10%]">Weight</TableHead>
            <TableHead className="w-[10%]">In-flight</TableHead>
            <TableHead className="w-[10%]">Breaker</TableHead>
            <TableHead className="w-[10%]">Enabled</TableHead>
            <TableHead className="w-[5%]"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {backends.map((backend, index) => (
            <TableRow key={index}>
              <TableCell className="font-medium">
                <p className="scroll-m-20 text-md font-semibold tracking-tight">{backend.name}</p>
                <p className="text-sm text-muted-foreground">{backend.baseUrl}</p>
              </TableCell>
              <TableCell className="flex flex-wrap gap-1">
                {backend.modelsServed.map((model) => (
                  <Badge key={model} variant="outline">{model}</Badge>
                ))}
              </TableCell>
              <TableCell>{backend.weight}</TableCell>
              <TableCell>{backend.inFlight}/{backend.total}</TableCell>
              <TableCell>
                <StatusPill status={backend.breakerState}>{backend.breakerState}</StatusPill>
              </TableCell>
              <TableCell>
                <div className="flex items-center space-x-2">
                  <Switch className="cursor-pointer" id={`${backend}-backend-enabled`} checked={backend.enabled}/>
                </div>
              </TableCell>
              <TableCell>
                <Button className="cursor-pointer" variant="secondary" size="icon" onClick={() => handleSelectBackend(backend.id)}>
                  <Edit />
                </Button>
              </TableCell>
            </TableRow>
          ))}

        </TableBody>
      </Table>
    </div>
  )
}
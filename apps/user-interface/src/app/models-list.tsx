import React from "react";
import { Plus, Edit } from "lucide-react";
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { StatusPill } from "@/components/ui/status-pill";
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { ModelForm } from "@/components/forms/model-form";

interface BackendProps {
  name: string;
  endpoint: string;
  models: string[];
  weight: number;
  inFlight: number;
  total: number;
  breakerState: "open" | "half-open" | "closed" | "disabled";
  enabled: boolean;
}

const backends: BackendProps[] = [
  {
    name: "vllm-a100-3",
    endpoint: "vllm-3.vllm.svc:8000",
    models: ["llama-70b", "llama-8b"],
    weight: 100,
    inFlight: 14,
    total: 64,
    breakerState: "closed",
    enabled: true,
  },
  {
    name: "vllm-a100-4",
    endpoint: "vllm-a100-4.vllm.svc:8000",
    models: ["llama-70b"],
    weight: 100,
    inFlight: 41,
    total: 64,
    breakerState: "half-open",
    enabled: true,
  },
  {
    name: "vllm-l40s-1",
    endpoint: "vllm-l40s-1.vllm.svc:8000",
    models: ["llama-8b", "qwen-coder"],
    weight: 50,
    inFlight: 0,
    total: 32,
    breakerState: "open",
    enabled: true,
  },
  {
    name: "vllm-3090-home",
    endpoint: "vllm-3090-home.vllm.svc:8000",
    models: ["llama-8b"],
    weight: 25,
    inFlight: 3,
    total: 16,
    breakerState: "closed",
    enabled: true,
  },
  {
    name: "vllm-a100-spare",
    endpoint: "vllm-a100-spare.vllm.svc:8000",
    models: ["llama-70b"],
    weight: 100,
    inFlight: 0,
    total: 0,
    breakerState: "disabled",
    enabled: false,
  },
]

export function ModelsList() {
  const [showModal, setShowModal] = React.useState(true);
  const toggleModal = () => setShowModal(!showModal);

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
                <p className="text-sm text-muted-foreground">{backend.endpoint}</p>
              </TableCell>
              <TableCell className="flex flex-wrap gap-1">
                {backend.models.map((model) => (
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
                <Button className="cursor-pointer" variant="secondary" size="icon">
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
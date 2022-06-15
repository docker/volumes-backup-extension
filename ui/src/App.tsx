import React, { useEffect } from "react";
import { DataGrid } from "@mui/x-data-grid";
import { createDockerDesktopClient } from "@docker/extension-api-client";
import { Stack, Button, Typography } from "@mui/material";

// Note: This line relies on Docker Desktop's presence as a host application.
// If you're running this React app in a browser, it won't work properly.
const client = createDockerDesktopClient();

function useDockerDesktopClient() {
  return client;
}

const columns = [
  { field: "id", headerName: "ID", width: 70, hide: true },
  { field: "volumeDriver", headerName: "Driver", width: 70 },
  { field: "volumeName", headerName: "Volume name", width: 260 },
  { field: "volumeMountPoint", headerName: "Mount point", width: 260 },
  { field: "volumeSize", headerName: "Size", width: 130 },
  {
    field: "export",
    headerName: "Action",
    width: 130,
    renderCell: (params) => (
      <Button
        variant="contained"
        onClick={() => console.log("exporting volume", params.row.volumeName)}
      >
        Export
      </Button>
    ),
  },
];

export function App() {
  const [rows, setRows] = React.useState([]);
  const ddClient = useDockerDesktopClient();

  useEffect(() => {
    const listVolumes = async () => {
      const result = await ddClient.docker.cli.exec("volume", [
        "ls",
        "--format",
        "'{{ json . }}'",
      ]);
      console.log(result);
      if (result.stderr !== "") {
        ddClient.desktopUI.toast.error(result.stderr);
      } else {
        const volumes = result.parseJsonLines();
        const rows = volumes.map((volume, index) => {
          return {
            id: index,
            volumeDriver: volume.Driver,
            volumeName: volume.Name,
            volumeMountPoint: volume.Mountpoint,
            volumeSize: volume.Size,
          };
        });

        setRows(rows);
      }
    };

    listVolumes();
  }, []); // run it once, only when component is mounted

  return (
    <>
      <Typography variant="h3">Vackup Extension</Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mt: 2 }}>
        Easily backup and restore docker volumes.
      </Typography>
      <Stack direction="row" alignItems="start" spacing={2} sx={{ mt: 4 }}>
        <div style={{ height: 400, width: "100%" }}>
          <DataGrid
            rows={rows}
            columns={columns}
            pageSize={5}
            rowsPerPageOptions={[5]}
            checkboxSelection={false}
          />
        </div>
      </Stack>
    </>
  );
}
